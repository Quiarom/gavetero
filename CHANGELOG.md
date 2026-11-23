# Changelog

All notable changes to Gavetero are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project
adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- `gvt ask --live --host <host>` queries a real router instead of the
  default mock fixture.

### Fixed

- `gvt ask` and `gvt inspect` pass the router password to the runtime over
  stdin when `--router-password-stdin` is set.
- `gvt doctor` reports the default gateway on Linux instead of the local
  interface address.
- The runtime keeps using the authenticated router adapter to serve
  observations after live login.

## [0.1.0-alpha.1] - 2026-09-05

This entry records the `gavetero-v0.1.0-alpha.1` source tag.

### Added

- The `gavetero` CLI (`gvt`) embeds the runtime and agent sidecars in one
  binary. It provides fixture-backed inspection, live router detection,
  setup and diagnostic commands, and questions via `gvt ask`.
- The read-only agent uses MiniMax M3 via GMI Cloud by default, with MiniMax
  M2.7 as a fallback for transient errors and a deterministic offline mode.
- The local API exposes a capability matrix at `/v0/capabilities` with four
  states: `verified`, `absent`, `unsupported_or_unverified`, and
  `unavailable`. Security endpoints report each capability independently.
- The agent serves `POST /v0/chat` and `GET /healthz` on loopback for the
  desktop frontend.
- The React and Tauri dashboard uses the documented JSON contract and mock
  fixtures.
- `gvt integrations install` installs the Gavetero skill for Hermes or
  OpenCode.
- Contributor and security guides, issue and pull request templates, the
  Contributor Covenant, and third-party attribution in `NOTICE`.

### Changed

- The TP-Link WR841N adapter uses the verified authentication recipe and
  reads firmware and hardware identifiers from the authenticated dashboard.

### Fixed

- Registered the local API handler for `/v0/clients`.
- Removed a duplicate slash in adapter request paths and stopped relying
  on firmware-specific array indices to parse the router identity.

## 2026-08-31 — Initial development snapshot

- Established separate domain, transport, and TP-Link WR841N adapter
  packages.
- Added the runtime (`router-core`) and the lab capture tool
  (`router-core-learn`).
- Added fixture-backed verification without hardware, Go CI checks, and
  the MIT license.

[Unreleased]: https://github.com/Quiarom/gavetero/compare/gavetero-v0.1.0-alpha.1...main
[0.1.0-alpha.1]: https://github.com/Quiarom/gavetero/tree/gavetero-v0.1.0-alpha.1
