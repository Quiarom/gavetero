package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Quiarom/gavetero/internal/domain"
)

func TestInspectObservations(t *testing.T) {
	for _, format := range []string{"json", "jsonl"} {
		for _, tc := range []struct {
			state  domain.ObservationState
			status int
			data   string
		}{
			{domain.Verified, http.StatusOK, `[{"ip":"192.0.2.1"}]`},
			{domain.Absent, http.StatusOK, "[]"},
			{domain.UnsupportedOrUnverified, http.StatusNotFound, "null"},
			{domain.Unavailable, http.StatusServiceUnavailable, "null"},
		} {
			t.Run(format+"/"+string(tc.state), func(t *testing.T) {
				clients := fmt.Sprintf(`{"state":%q,"reason":"test reason","provenance":"fixture","data":%s}`, tc.state, tc.data)
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/v0/clients" {
						w.WriteHeader(tc.status)
						fmt.Fprint(w, clients)
						return
					}
					fmt.Fprint(w, `{"state":"verified","reason":"test reason","provenance":"fixture","data":{"observed":true}}`)
				}))
				defer srv.Close()
				var out bytes.Buffer
				err := inspectObservations(t.Context(), &out, srv.Client(), srv.URL, inspectOptions{Output: format})
				if (err != nil) != (tc.state == domain.Unavailable) {
					t.Fatalf("unexpected inspect error: %v", err)
				}
				var result map[string]json.RawMessage
				if err := json.Unmarshal(out.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				var compact bytes.Buffer
				if err := json.Compact(&compact, result["clients"]); err != nil || compact.String() != clients {
					t.Fatalf("clients = %s (%v), want %s", result["clients"], err, clients)
				}
			})
		}
	}
}

func TestRenderInspectHuman(t *testing.T) {
	for _, live := range []bool{false, true} {
		for _, state := range []domain.ObservationState{domain.Verified, domain.Absent, domain.UnsupportedOrUnverified, domain.Unavailable} {
			t.Run(fmt.Sprintf("live=%t/%s", live, state), func(t *testing.T) {
				observations := map[string]domain.Observation[json.RawMessage]{}
				for _, name := range []string{"device", "status", "clients", "capabilities"} {
					observations[name] = domain.Observation[json.RawMessage]{
						State: state, Reason: "test reason", Provenance: domain.ProvenanceFixture, Data: json.RawMessage("null"),
					}
				}
				observations["capabilities"] = domain.Observation[json.RawMessage]{
					State: domain.Verified, Reason: "test reason", Provenance: domain.ProvenanceFixture,
					Data: json.RawMessage(`{"device":"verified","extra":"verified"}`),
				}
				var out bytes.Buffer
				if err := renderInspectHuman(&out, observations, inspectOptions{Live: live, Host: "192.0.2.1"}); err != nil {
					t.Fatal(err)
				}
				if live {
					if !strings.Contains(out.String(), "device: "+string(state)+" (test reason; fixture)") || strings.Contains(out.String(), "Read your real router:") {
						t.Fatalf("live presentation changed: %s", out.String())
					}
					return
				}
				label := "mock/" + string(state)
				if state == domain.Verified {
					label = "mock"
				}
				for _, name := range []string{"device", "status", "clients"} {
					if !strings.Contains(out.String(), name+": "+label+" (test reason; fixture)") {
						t.Fatalf("unlabeled demo observation: %s", out.String())
					}
				}
				for _, key := range []string{"device", "extra"} {
					if !strings.Contains(out.String(), fmt.Sprintf("  %-22s mock\n", key)) {
						t.Fatalf("unlabeled demo capability: %s", out.String())
					}
				}
				if !strings.Contains(out.String(), "mock (test data, not a real router)") || strings.Contains(out.String(), ": verified") || !strings.HasSuffix(out.String(), "Read your real router: gvt inspect --live --host <router-ip>\n") {
					t.Fatalf("misleading demo output: %s", out.String())
				}
			})
		}
	}
	t.Run("malformed data", func(t *testing.T) {
		observations := map[string]domain.Observation[json.RawMessage]{
			"device":       {Data: json.RawMessage(`{`)},
			"status":       {Data: json.RawMessage(`null`)},
			"clients":      {Data: json.RawMessage(`null`)},
			"capabilities": {Data: json.RawMessage(`null`)},
		}
		if err := renderInspectHuman(&bytes.Buffer{}, observations, inspectOptions{}); err == nil || !strings.Contains(err.Error(), "decode device data") {
			t.Fatalf("expected decode error: %v", err)
		}
	})
}

func TestInspectObservations_MockJSON(t *testing.T) {
	for _, format := range []string{"json", "jsonl"} {
		for _, live := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/live=%t", format, live), func(t *testing.T) {
				state, provenance, mode := "mock", "fixture", `,"mode":"mock"`
				if live {
					state, provenance, mode = "verified", "observed", ""
				}
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					fmt.Fprintf(w, `{"state":%q,"reason":"read","provenance":%q,"data":{"present":true}%s}`, state, provenance, mode)
				}))
				defer srv.Close()
				var out bytes.Buffer
				if err := inspectObservations(t.Context(), &out, srv.Client(), srv.URL, inspectOptions{Output: format, Live: live, Host: "192.0.2.1"}); err != nil {
					t.Fatal(err)
				}
				var result map[string]any
				if err := json.Unmarshal(out.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if live {
					if result["mode"] != "live" || result["provenance"] != nil || strings.Contains(out.String(), `"mock"`) {
						t.Fatalf("live context changed: %s", out.String())
					}
				} else if result["mode"] != "mock" || result["provenance"] != "fixture" || strings.Contains(out.String(), `"verified"`) {
					t.Fatalf("incorrect mock context: %s", out.String())
				}
			})
		}
	}
}
