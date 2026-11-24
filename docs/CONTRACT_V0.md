# Gavetero observation contract: /v0 (DRAFT)

This draft describes the current read-only observation interface. It is not a
stability commitment. The decision to stabilize `/v0` or include Wi-Fi in the
alpha is pending.

## Observation envelope

Each observation has the same four keys in HTTP, CLI machine output, and agent
tool results. Use `state` for decisions and `reason` for display; reason text
is not a machine-readable code.

```json
{
  "state": "absent",
  "reason": "no DHCP lease rows were observed",
  "provenance": "observed",
  "data": []
}
```

| State | What it means | HTTP on a known GET route |
| --- | --- | --- |
| `verified` | The named source yielded at least one usable fact. Known `"false"` and numeric zero count as data. Some fields may still be unknown. | 200 |
| `absent` | The source was missing or yielded no observation. This is not proof that the feature is off or that the firmware never implements it. | 200 |
| `unsupported_or_unverified` | There is no verified support for this observation. Provisional Wi-Fi data may be present, but must not be treated as verified. | 404 without data, 200 with provisional data |
| `unavailable` | The observation could not be obtained or interpreted (for example, a session, transport or parsing failure). | 503 |

`provenance` is `observed` for a router response, `fixture` for replay, and
`absent` when there is no recorded source. Mock mode reads fixture files
relative to the working directory; they are not embedded. If a fixture file
is missing, its observation is `absent` with `provenance: "absent"` and
`data: null`. A file successfully replayed with an empty DHCP lease list
has `state: "absent"`, `provenance: "fixture"` and `data: []`.
Verified fixture data confirms only what the replay produced;
**fixtures are not evidence of current hardware**.
When no data exists the field is `null`, except an empty DHCP lease list may
be `[]`. **Not observed does not mean off**; missing fields, `"unknown"`, and
`null` must not be interpreted as false or zero.

## Read-only endpoints and tools

| GET path | Agent tool | `data` |
| --- | --- | --- |
| `/v0/device` | `get_device_info` | Normalized device identity and authentication |
| `/v0/status` | `get_router_status` | Reachability, WAN link state and uptime; not a speed test |
| `/v0/clients` | `get_clients` | Array of DHCP lease rows, not a list of every device. It does not prove active presence or Wi-Fi association and does not inventory static-IP hosts. |
| `/v0/capabilities` | `get_capabilities` | Map of capability names to states, including existing aliases. A verified matrix means the summary is present, not that its individual entries are verified. |
| `/v0/security/wireless` | `get_wireless_security` | Provisional allowlist: `enabled`, `ssid`, `securityTypeRaw`, `securityType`, `cipher`, `keyRenewalSecs`, `hasPreSharedKey`. No raw array, key, or PIN. Only historical evidence exists on firmware 3.15.9. |
| `/v0/security/wps` | `get_wps_state` | WPS enabled state |
| `/v0/security/dmz` | `get_dmz_state` | DMZ enabled state and destination, when observed |
| `/v0/security/upnp` | `get_upnp_state` | UPnP enabled state and mapping count, when observed |
| `/v0/security/remote-management` | `get_remote_management_state` | Remote management enabled state and port, when observed |
| `/v0/security/forwarding` | `get_forwarding_rules` | Count of forwarding rules, not externally reachable services |

The observation routes accept GET only. A wrong method on a known route returns
405 with an unavailable envelope. `/healthz` and the agent's `/v0/chat` are
liveness and orchestration endpoints, not router observations.

`gvt inspect --output json` includes the envelopes in `device`, `status`,
`clients`, and `capabilities`. JSONL includes the same observations; the
identity and matrix keys there are `router` and `caps`. A valid unavailable
result is printed and makes inspect exit nonzero; absent and unsupported
observations are not command errors. Invalid responses and command errors also
exit nonzero.

Agent tools pass the HTTP envelope unchanged to the model and expose it as
`steps[].result` in JSON chat output. HTTP status and tool name are separate
metadata. The live agent uses an unavailable envelope if it cannot get a
valid router-core response; it never infers verification from HTTP 200.
