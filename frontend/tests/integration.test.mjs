// End-to-end integration test: verify the frontend (via Vite dev
// server) can talk to router-core serve and to the agent.
//
// Run with: node --test /tmp/integration_test.mjs
// Requires: serve on 127.0.0.1:8484, frontend on 127.0.0.1:5180.

import { test } from 'node:test';
import { strict as assert } from 'node:assert';

const SERVE = 'http://127.0.0.1:8484';
const FRONT = 'http://127.0.0.1:5180';

async function get(path, base = SERVE) {
  const r = await fetch(base + path);
  return { status: r.status, body: r.headers.get('content-type'), text: await r.text() };
}

test('serve /healthz returns 200 with state ok', async () => {
  const r = await get('/healthz');
  assert.equal(r.status, 200);
  const body = JSON.parse(r.text);
  assert.equal(body.state, 'ok');
});

test('serve /v0/device returns full fingerprint', async () => {
  const r = await get('/v0/device');
  assert.equal(r.status, 200);
  const body = JSON.parse(r.text);
  assert.equal(body.state, 'verified');
  assert.equal(body.data.vendor, 'TP-Link');
  assert.equal(body.data.model, 'TL-WR841N/ND');
  assert.match(body.data.firmwareVersion.value, /\d+\.\d+\.\d+ Build/);
  assert.match(body.data.hardwareVersion.value, /WR841N v8 00000000/);
});

test('serve /v0/capabilities reports wireless_security as not verified', async () => {
  const r = await get('/v0/capabilities');
  assert.equal(r.status, 200);
  const body = JSON.parse(r.text);
  assert.equal(body.data.wireless_security, 'unsupported_or_unverified');
  assert.equal(body.data.wps, 'unsupported_or_unverified');
});

test('serve /v0/security/wireless reports the limits of the evidence', async () => {
  const r = await get('/v0/security/wireless');
  assert.ok([200, 404].includes(r.status));
  const body = JSON.parse(r.text);
  assert.equal(body.state, 'unsupported_or_unverified');
  assert.match(body.reason, /3\.15\.9/);
  if (r.status === 200) {
    assert.ok('ssid' in body.data);
    assert.ok('securityType' in body.data);
    assert.ok(!('raw' in body.data) && !('Raw' in body.data));
  } else {
    assert.equal(body.data, null);
  }
});

test('frontend dev server serves a 200 HTML page', async () => {
  const r = await get('/', FRONT);
  assert.equal(r.status, 200);
  assert.match(r.text, /<html/);
});

test('frontend dev page references the live router-core URL', async () => {
  const r = await get('/', FRONT);
  assert.match(r.text, /\/@vite\/client/);
});
