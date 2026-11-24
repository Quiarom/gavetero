package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quiarom/gavetero/internal/adapters/fixture"
	"github.com/Quiarom/gavetero/internal/domain"
)

func TestRegisterRoutes_Mock(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocked, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"", t.TempDir(), blocked} {
		t.Run(dir, func(t *testing.T) {
			mux := http.NewServeMux()
			registerRoutes(mux, fixture.New(dir), nil)
			for _, path := range []string{"device", "status", "clients", "capabilities", "security/wireless", "security/wps", "security/dmz", "security/upnp", "security/remote-management", "security/forwarding"} {
				for _, method := range []string{http.MethodGet, http.MethodPost} {
					w := httptest.NewRecorder()
					mux.ServeHTTP(w, httptest.NewRequest(method, "/v0/"+path, nil))
					got, err := domain.DecodeObservation(w.Body.Bytes())
					if err != nil || got.Mode != "mock" || got.Provenance != domain.ProvenanceFixture || strings.Contains(w.Body.String(), `"verified"`) {
						t.Fatalf("%s %s: %s (%v)", method, path, w.Body.String(), err)
					}
					if method == http.MethodPost && (w.Code != http.StatusMethodNotAllowed || got.State != domain.Unavailable) {
						t.Fatalf("method rejection changed: %d %s", w.Code, w.Body.String())
					}
					if method == http.MethodGet && path == "device" {
						want := domain.Mock
						if dir == blocked {
							want = domain.Unavailable
						} else if dir != "" {
							want = domain.Absent
						}
						if got.State != want {
							t.Fatalf("device state=%s want=%s", got.State, want)
						}
					}
				}
			}
		})
	}
}

func TestObservationHandler_LiveEncoding(t *testing.T) {
	w := httptest.NewRecorder()
	observationHandler(func(context.Context) domain.Observation[any] {
		return observed("read", domain.ProvenanceObserved, map[string]bool{"present": true})
	})(w, httptest.NewRequest(http.MethodGet, "/v0/device", nil))
	want := "{\"state\":\"verified\",\"reason\":\"read\",\"provenance\":\"observed\",\"data\":{\"present\":true}}\n"
	if w.Body.String() != want || w.Code != http.StatusOK {
		t.Fatalf("live encoding changed: %d %s", w.Code, w.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result["mode"] != nil {
		t.Fatalf("live gained mock metadata: %v (%v)", result, err)
	}
}
