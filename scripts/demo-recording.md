# Recording the Gavetero demo

The demo shows gavetero working against a real
router. The router the user controls is at
192.168.1.1 (TP-Link, default credentials are
admin/admin or whatever the user has set).

## Capture

The recommended tool on Omarchy (Wayland-based
Arch) is `wf-recorder`:

```sh
# Screen + audio, save to /tmp/demo.mp4
wf-recorder -a -f /tmp/demo.mp4

# Convert to web-friendly mp4
ffmpeg -i /tmp/demo.mp4 -c:v libx264 -preset slow \
  -crf 22 -c:a aac -b:a 128k /tmp/demo-final.mp4
```

## Demo script (3 minutes)

1. `gvt doctor` (10s)
   Shows 6/6 checks pass. This proves the install
   is clean: CLI, config, credential, network,
   router, adapter.

2. `gvt detect 192.168.1.1` (10s)
   Shows the family hint (tplink-userrpm) and
   that the device is reachable. This is the
   "any-router path" working.

3. `gvt inspect --live --host 192.168.1.1
   --router-password-stdin` (15s)
   The sidecar spawns. If the user knows the
   password, the inspect returns the live
   observations: 4-state vocabulary, real
   device info, real capabilities. If the user
   does not know the password, the demo shows
   the honest timeout (the sidecar returned
   401, gavetero reports it). Both are correct.

4. `gvt ask "Is my Wi-Fi exposed?" --dry-run`
   (20s)
   Shows the stub agent in mock mode. The user
   can show the structured-events format here.

5. `gvt ask "Is my Wi-Fi exposed?"` with
   `GMI_SERVING_API_KEY=<key>` exported (45s)
   This is the real thing. The agent makes 9 tool
   calls (4 verified, 5 unsupported_or_unverified),
   produces a Spanish answer with the four-state
   vocabulary, states the evidence limits.

6. `gvt integrations install opencode` (5s)
   The skill lands at
   `~/.config/opencode/skills/gavetero/SKILL.md`.

7. (Optional) Open Hermes/OMP in a new session,
   type `/gavetero investigate my network`. Show
   that the skill is loaded and the M3 agent uses
   it.

## Notes

- The `gvt inspect` demo does NOT require the
  real router password. The user is the
  authoritative source. If they know the
  password, they can pipe it via stdin. If they
  do not, the demo shows the honest timeout.
- All commands are deterministic and
  reproducible.
- The recording is plain shell output. No
  special camera work is needed.
