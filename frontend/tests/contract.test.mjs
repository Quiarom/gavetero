// Contract smoke test for the router-core frontend.
//
// This test reads the live OpenRouter trace fixture captured
// against the real lab unit and asserts that the per-turn
// shape matches what the frontend (AiAssistantView, Dashboard)
// expects to consume. It runs with the built-in `node --test`
// runner, so there is no extra dependency to install.
//
// Run it from the repo root:
//
//   node --test frontend/tests/contract.test.mjs
//
// The frontend dev can wire this into their own CI alongside
// vitest. Dashboard regressions render React using the existing
// Vite and React dependencies; no additional runner is needed.

import { after, before, describe, test } from "node:test";
import { readFileSync, readFileSync as _r } from "node:fs";
import { strict as assert } from "node:assert";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { createServer } from "vite";

const __dirname = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(__dirname, "..", "..");
const TRACE = resolve(REPO_ROOT, "fixtures", "agent-traces", "2026-09-04-wifi-exposed.live.jsonl");

function loadTrace() {
  const text = readFileSync(TRACE, "utf8");
  const lines = text.split("\n").filter((l) => l.trim() !== "");
  return lines.map((l) => JSON.parse(l));
}

test("live trace exists and is non-empty", () => {
  const turns = loadTrace();
  assert.ok(turns.length >= 4, `expected >=4 turns, got ${turns.length}`);
});

test("turn 1 is the user question", () => {
  const t = loadTrace()[0];
  assert.equal(t.role, "user");
  assert.equal(t.content, "Is my Wi-Fi exposed?");
});

test("turn 2 is the assistant tool call for wireless", () => {
  const t = loadTrace()[1];
  assert.equal(t.role, "assistant");
  assert.ok(Array.isArray(t.tool_calls), "tool_calls must be an array");
  assert.equal(t.tool_calls.length, 1);
  const tc = t.tool_calls[0];
  assert.equal(tc.function.name, "get_security");
  // The frontend posts {question: "..."} to /v0/chat and the agent
  // dispatches the tool with {name: "..."}. Assert the agent
  // chose the right capability.
  const args = JSON.parse(tc.function.arguments);
  assert.equal(args.name, "wireless");
});

test("turn 3 is the tool result with the parsed wireless body", () => {
  const t = loadTrace()[2];
  assert.equal(t.role, "tool");
  assert.equal(t.name, "get_security");
  // The tool content is a JSON string. The router-core serve
  // wraps the parsed WirelessSecurity struct in {result: ...}.
  const body = JSON.parse(t.content);
  assert.equal(body.result.SSID, "TP-LINK_CBEC16");
  assert.equal(body.result.SecurityType, "wpa2-psk");
  assert.equal(body.result.Cipher, "332");
  assert.ok(body.result.HasPreSharedKey, "HasPreSharedKey must be true");
});

test("turn 4 is the final assistant answer, in Spanish, with a recommendation", () => {
  const t = loadTrace()[3];
  assert.equal(t.role, "assistant");
  assert.ok(t.content.length > 100, "answer must be substantive");
  // Spot-check Spanish content + a recommendation.
  assert.ok(/recomendaci[oó]n/i.test(t.content), "answer must include a recommendation");
  assert.ok(/WPA2|wpa2/i.test(t.content), "answer must reference WPA2");
});

describe("honest dashboard observations", () => {
  let server;
  let DashboardView;
  let CapabilitiesGrid;
  let capabilityStateText;

  before(async () => {
    server = await createServer({
      root: resolve(__dirname, ".."),
      server: { middlewareMode: true, watch: null, ws: false }
    });
    ({ DashboardView } = await server.ssrLoadModule("/src/components/Dashboard.jsx"));
    ({ CapabilitiesGrid, capabilityStateText } = await server.ssrLoadModule("/src/components/CapabilitiesGrid.jsx"));
  });
  after(async () => {
    await server?.close();
  });

  const response = (status, state, provenance, data, reason = state) => ({
    status, data: { state, reason, provenance, data }
  });
  const dashboard = (observations, isLive = true) =>
    renderToStaticMarkup(createElement(DashboardView, { observations, isLive }));

  test("DHCP 503 and a missing lease source do not render as an observed empty list", () => {
    for (const clients of [
      response(503, "unavailable", "absent", null),
      response(200, "absent", "absent", null)
    ]) {
      const html = dashboard({ clients });
      assert.match(html, /Concesiones DHCP no disponibles/);
      assert.doesNotMatch(html, /0 CONCESIONES DHCP/);
    }
    const html = dashboard({ clients: response(200, "absent", "observed", []) });
    assert.match(html, /0 CONCESIONES DHCP/);
    assert.match(html, /La lista DHCP observada está vacía/);
  });

  test("reachability does not imply a connected WAN; unknown is not disconnected", () => {
    for (const [wanStatus, label] of [
      ["disconnected", "WAN DESCONECTADA"],
      ["unknown", "WAN NO DISPONIBLE"],
      ["connected", "WAN CONECTADA"]
    ]) {
      const status = response(200, "verified", "observed", { reachable: "true", wanStatus });
      const html = dashboard({ status });
      assert.match(html, new RegExp(label));
      assert.doesNotMatch(html, /INTERNET CONECTADO|SIN CONEXIÓN/);
    }
  });

  test("fixture leases do not imply client link type, signal, or current connectivity", () => {
    const clients = response(200, "verified", "fixture", [
      { name: { value: "pixel-phone" }, ip: "192.168.1.100", mac: "00:01:02:03:04:05" }
    ]);
    const html = dashboard({ clients });
    assert.match(html, /pixel-phone/);
    assert.match(html, /1 CONCESIONES DHCP/);
    assert.match(html, /datos de fixture, no evidencia de hardware/);
    assert.doesNotMatch(html, /Señal|title="Conectado"|>CABLE</);
  });

  test("unverified security cards make no claims about configuration", () => {
    const html = renderToStaticMarkup(createElement(CapabilitiesGrid, {
      capabilities: { dmz: "unsupported_or_unverified", forwarding: "unavailable" }
    }));
    assert.match(html, /POR VERIFICAR/);
    assert.match(html, /NO DISPONIBLE/);
    assert.doesNotMatch(html, /disabled|0 active|apagada|hardware real/);
    assert.match(capabilityStateText("verified", "fixture"), /Datos de fixture; no son evidencia de hardware/);
    assert.match(capabilityStateText("absent", "absent"), /no implica que la función esté desactivada/);
  });

  test("mock view does not render errors from local service observations", () => {
    const observations = { device: response(503, "unavailable", "absent", null, "Falló la lectura de identidad") };
    assert.match(dashboard(observations), /Falló la lectura de identidad/);
    assert.doesNotMatch(dashboard(observations, false), /role="alert"|Falló la lectura de identidad/);
  });
});
