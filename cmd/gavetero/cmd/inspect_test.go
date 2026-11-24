package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
