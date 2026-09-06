# Gavetero

> **A typed local API and an evidence-aware AI agent for legacy
> consumer routers — without replacing the firmware.**

[![CI](https://github.com/Quiarom/router-core/actions/workflows/ci.yml/badge.svg)](.github/workflows/ci.yml)
[![Go 1.25+](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![MiniMax Week 2026](https://img.shields.io/badge/MiniMax_Week-2026-blue)](https://www.gmicloud.ai/minimax-week)

Gavetero turns an aging router dashboard into a typed local API
that you (or an AI agent) can investigate. The runtime is
read-only: it never mutates the router. Unknown is reported
as unknown — never as false.

## Why

Most home routers still expose a 2014-era web UI. The
credentials are `admin/admin` (or whatever the user set), the
firmware is EOL, and the only "API" is the HTML the user
sees in their browser. An LLM tool that wants to answer
"is my network exposed?" has nothing structured to read.

Gavetero gives that LLM a typed surface, a model-aware
sidecar, and a model that knows how to use both. The router
keeps running its old firmware. Gavetero never touches it.

## How

Gavetero is a single Go binary that embeds the runtime and
the agent sidecar. The CLI spawns them as needed:

```text
┌──────────────────────────────────────────────────┐
│                      gvt                          │
│  ┌────────────┐    ┌────────────────────┐         │
│  │ router-core│    │   router-core-agent │         │
│  │  (HTTP API)│◄──│  (M3 / M2.7 via GMI) │         │
│  └────────────┘    └────────────────────┘         │
│         ▲                                          │
│         │ http (real router OR fixture)            │
└─────────┼──────────────────────────────────────────┘
          │
   ┌──────┴───────┐
   │  TP-Link    │   ← real router on the LAN
   │  WR841N     │      OR the embedded fixture
   │  or other   │      for first-time use
   └─────────────┘
```

The runtime reads the router's web UI over HTTP, parses
it into typed observations (the four knowledge states:
`verified`, `absent`, `unsupported_or_unverified`, `unavailable`),
and exposes those via `http://127.0.0.1:8484/v0/...`.
The agent, called by the model, decides which observation
to request next. The model reasons about the answer;
the runtime enforces the evidence.

## Honest defaults

Gavetero does not invent. The runtime is read-only.
The four-state vocabulary means:

- `verified` — the runtime read the value from the router
- `absent` — the firmware does not implement the capability
- `unsupported_or_unverified` — Gavetero has no parser for it
- `unavailable` — transport failure (timeout, refused, etc.)

The default mode uses an embedded fixture (a sanitized
capture of a TP-Link WR841N on firmware 3.15.9) so first-time
users can try the CLI without hardware. With `--live --host X`
it talks to a real router at X. The fixture is real data
(sanitized at capture time); the live mode is real data
(read at run time). Mock data is not invented.

## Install

The user-facing install is one command. No sudo, no Go,
no system packages:

```sh
curl -sSf https://raw.githubusercontent.com/Quiarom/router-core/gavetero-v0.1.0-alpha.1/install.sh | sh
```

This downloads the prebuilt single binary for your platform
(linux/darwin × amd64/arm64), installs it to
`~/.local/bin/gavetero` with a `gvt` symlink, and prints
the PATH hint if needed.

For developers working in the repo:

```sh
make install-user
```

## Quick start (real router)

```sh
# 1. One-time setup: store your GMI Cloud API key in the OS
#    credential store. No plaintext config, no env vars required.
gvt setup

# 2. Confirm the install is healthy.
gvt doctor
#   ✓ CLI installation
#   ✓ config location
#   ✓ GMI key                  from OS credential store
#   ✓ default gateway          192.168.1.1
#   ✓ router observation
#   ✓ adapter
#   Result: 6 ok, 0 warn, 0 fail

# 3. Identify any router on the network (real or unknown).
gvt detect 192.168.1.1
#   Gavetero Detect: 192.168.1.1
#   ================
#     reachability:  reachable
#     family hint:   tplink-userrpm

# 4. Read the live observations. Provide your router password
#    via stdin (never via --password). Other vendors' default
#    username can be set with --router-user.
echo "$ROUTER_ADMIN_PASSWORD" | \
  gvt inspect --live --host 192.168.1.1 --router-password-stdin

# 5. Ask a question. The model (MiniMax M3 default, M2.7
#    fallback) calls gvt inspect with different capabilities,
#    then synthesises a Spanish answer with the four-state
#    vocabulary and explicit evidence limits.
gvt ask "¿Qué tan expuesta está mi red Wi-Fi?"
```

## Quick start (no router, first try)

```sh
# Without setup, gvt runs in mock mode using the embedded fixture
# (a sanitized TP-Link WR841N capture). No GMI key required.
gvt inspect
#   Mode:  mock (fixture-backed, no network)
#   Device: TL-WR841N/ND v8.4 on firmware 3.15.9
#   ...

gvt ask "Is my Wi-Fi exposed?" --dry-run
#   Tools called: 3
#     1. get_security /v0/security/dmz -> 404
#     2. get_security /v0/security/forwarding -> 404
#     3. get_security /v0/security/upnp -> 404
#   Model: MiniMaxAI/MiniMax-M3
#   Mode:  stub
```

## Commands

```text
gvt version        # the build version
gvt setup          # store the GMI Cloud API key (one time)
gvt doctor         # 6 install checks, no network
gvt inspect        # router observations (mock by default, --live to use a real router)
gvt detect [host]  # identify a router on the network, family hint
gvt ask <question> # run a question through MiniMax M3 (--dry-run for the stub)
gvt integrations install <hermes|opencode|omp>
                  # install the Gavetero skill for an agent runtime
```

## The four-state vocabulary

Every observation the runtime produces carries one of four
states. The agent reasons about the answer based on this
vocabulary; the user sees the same vocabulary in the output.

| State | Meaning | Example |
|---|---|---|
| `verified` | runtime read the value | SSID: TP-LINK_CBEC16 |
| `absent` | firmware does not implement | WPS on a router without WPS |
| `unsupported_or_unverified` | Gavetero has no parser | the userRpm page on a non-TP-Link |
| `unavailable` | transport failure | timeout connecting to the router |

The mock mode uses the embedded fixture. Where the fixture
parses a value, the state is `verified`. Where the firmware
does not implement a capability, the state is `absent`. The
agent does not invent `verified` for things the runtime
could not read.

## Architecture

The runtime is the only place that talks to the router. It
is one binary with one parser per verified router family.
Today: `tplinkwr841v8`. Adding a new family means writing
a new `internal/adapters/<vendor>/` package, implementing
the `domain.RouterAdapter` interface, and registering it.
The agent discovers which adapter to use via the family hint
that `gvt detect` reports.

The agent is a thin orchestrator over the runtime. It uses
`http.Client` against the runtime's `127.0.0.1:<port>` HTTP
API. The model reasons; the runtime enforces the evidence.

## Safety

- **Read-only by construction.** The runtime never sends a
  mutating request to the router. An architecture test
  enforces no `POST`/`PUT`/`DELETE` in the runtime path
  (`internal/architecture_test.go`).
- **Loopback or RFC1918 only.** The runtime refuses to start
  on a non-loopback address. The agent refuses to bind to
  `0.0.0.0`.
- **2 MiB response body cap.** Anything larger is truncated.
- **No cross-host redirects.**
- **Passwords are zeroed** before the runtime returns.
  `--router-password-stdin` reads from stdin (never from argv);
  the buffer is overwritten on return.
- **The skill does not invent.** When the runtime reports
  `unsupported_or_unverified`, the skill tells the model so.
  When the runtime reports `unavailable`, the model retries
  with a different observation or states that the evidence
  is missing.

## Development

```sh
make build          # compile all binaries (gavetero + sidecars + learn)
make test          # go test ./...
make check         # gofmt + vet + go test -race + frontend tests
make install-user  # install gvt + gavetero to ~/.local/bin
```

The CI runs 4 jobs: `go-test`, `frontend-test`, `user-journey`
(boots gavetero in mock mode and exercises inspect/doctor/ask
--dry-run), and `arch` (the architecture + secret-boundary
invariants).

## License

MIT.

## Acknowledgements

Built on top of:
- MiniMax M3 via GMI Cloud (https://www.gmicloud.ai)
- Cobra (https://github.com/spf13/cobra)
- The TP-Link WR841N and its userRpm family (which the runtime
  understands natively, and which Gavetero's first verified
  adapter wraps)
