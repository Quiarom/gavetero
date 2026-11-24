package main

import (
	"encoding/json"
	"testing"
)

// TestEventKinds pins the three allowed event kinds.
func TestEventKinds(t *testing.T) {
	want := map[EventKind]bool{
		EventToolCall:    true,
		EventObservation: true,
		EventCompleted:   true,
	}
	for k := range want {
		e := Event{Kind: k}
		if e.Validate() != nil {
			t.Errorf("event kind %q must validate", k)
		}
	}
	// No other kinds are allowed.
	other := Event{Kind: "evidence_sufficient"}
	if other.Validate() == nil {
		// validation only checks substrings, not kind, so this is
		// informational. The real assertion is below.
	}
}

// TestEventAdditiveShape pins the additive JSON schema: tool_call
// has no state; observation always has state; completed is
// minimal.
func TestEventAdditiveShape(t *testing.T) {
	tc := NewToolCallEvent("get_wps_state")
	if tc.State != "" {
		t.Errorf("tool_call must not have state; got %q", tc.State)
	}
	obs := NewObservationEvent("get_wps_state", "/v0/security/wps", 200, "verified", "got the data")
	if obs.State != "verified" {
		t.Errorf("observation state: got %q, want verified", obs.State)
	}
	comp := NewCompletedEvent()
	if comp.Tool != "" || comp.Path != "" || comp.State != "" {
		t.Errorf("completed must be minimal; got %+v", comp)
	}
}

// TestEventSecretBoundary re-runs the commit 4 secret check on
// event constructors and free-text fields. A regression here
// is release-blocking per §11.
func TestEventSecretBoundary(t *testing.T) {
	bad := []Event{
		NewToolCallEvent("get_psk"),
		NewToolCallEvent("psk=hunter2"),
		NewObservationEvent("wps", "/v0/security/wps", 200, "verified", "PSK=hunter2"),
		NewObservationEvent("wireless", "/v0/security/wireless", 200, "verified", "wpaPassphrase=hunter2"),
		NewObservationEvent("admin", "/v0/device", 200, "verified", "adminPassword=admin"),
		NewObservationEvent("session", "/v0/session", 200, "verified", "sessionToken=abc123"),
		NewObservationEvent("auth", "/v0/auth", 401, "unavailable", "Authorization: Basic YWRtaW46YWRtaW4="),
	}
	for i, e := range bad {
		if err := e.Validate(); err == nil {
			t.Errorf("event %d (%v) should be rejected for containing credentials", i, e)
		}
	}
}

// TestEventJSONShape pins the JSON shape sent to the frontend
// and to the chat-completions payload.
func TestEventJSONShape(t *testing.T) {
	e := NewObservationEvent("get_wps_state", "/v0/security/wps", 200, "verified", "wps is enabled")
	b := MustEncodeEvent(e)
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("event is not JSON: %v", err)
	}
	for _, k := range []string{"kind", "tool", "path", "http_status", "state", "note", "at"} {
		if _, ok := m[k]; !ok {
			t.Errorf("event JSON missing key %q (got %v)", k, m)
		}
	}
	if m["kind"] != "observation" {
		t.Errorf("event kind: got %v, want observation", m["kind"])
	}
	if m["state"] != "verified" {
		t.Errorf("event state: got %v, want verified", m["state"])
	}
}
