# ADR 0006: Sercomm read-only adapter and RPC transport

## Context

The initial router-core implementation supported only the legacy TP-Link TL-WR841N v8.4, which used HTTP Basic Auth over GET and returned server-rendered HTML containing JavaScript array definitions.

Modern ISP gateways, such as the Sercomm AT904X (and related Sercomm/Sagemcom firmware builds), use a single-page Vue.js application. Identification parameters (`fw_version`, `wan_ip4_addr`, `encryption_key`, `salt`) are exposed via `GET /data/user_lang.json`. However, subsequent authenticated observation requires:

1. A challenge-response login via `POST /data/login.json` using HMAC-SHA256 with a dynamic salt and key.
2. Observation queries dispatched via `POST /data/data.cgi` containing JSON-RPC 2.0 payloads with `"method": "GET"`.

## Decision

1. **Introduce `internal/adapters/sercomm`:** Implement `domain.RouterAdapter` for Sercomm gateways.
2. **Read-Only RPC Boundary:** The Sercomm adapter is authorized to issue `POST /data/login.json` (for session establishment) and `POST /data/data.cgi` (for JSON-RPC read queries).
3. **Safety Invariant Preserved:** Mutating operations (`POST /data/reset.json`, firmware update, settings mutation) remain strictly forbidden and unrepresentable. Only JSON-RPC queries with `"method": "GET"` for read-only parameter namespaces (`scinfo.@wan`, `wireless.general`, `hosts.@host`) are issued.
4. **Adapter Auto-Detection:** `router-core serve` probes `/data/user_lang.json` on the target host. If the Sercomm fingerprint is detected, the Sercomm adapter is instantiated; otherwise, it defaults to the verified TP-Link adapter.
