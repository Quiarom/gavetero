package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Quiarom/gavetero/internal/adapters/tplinkwr841v8"
	"github.com/Quiarom/gavetero/internal/domain"
	"github.com/Quiarom/gavetero/internal/transport"
)

// wr841nServerForServe emulates the WR841N v8.4 firmware for the
// serve binary's runtime tests. The Basic Auth recipe matches
// ADR 0005: plaintext password over the Authorization header.
// /userRpm/StatusRpm.htm returns the dashboard body only when
// the Referer header matches the parent frameset URL, the recipe
// verified live 2026-08-31.
type wr841nServerForServe struct {
	server *httptest.Server
}

func newWR841NForServe(t *testing.T, user, pass string) *wr841nServerForServe {
	t.Helper()
	authValue := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
	expected := "Basic " + authValue

	mux := http.NewServeMux()
	mux.HandleFunc("/userRpm/WlanSecurityRpm.htm", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != expected || r.Header.Get("Referer") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = io.WriteString(w, `<script>var wlanPara = new Array(8,0,3,"33",0,0,0,1800,0,"<TEST_PSK>"); var ssidList = new Array("test-network");</script>`)
	})
	mux.HandleFunc("/userRpm/AssignedIpAddrListRpm.htm", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != expected {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, `<html><body><script>var DHCPDynList = new Array("omarchy", "00:11:22:33:44:55", "192.168.1.100", "01:24:35", 0, 0); var DHCPDynPara = new Array(1, 4, 0, 0);</script></body></html>`)
	})
	mux.HandleFunc("/userRpm/StatusRpm.htm", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != expected {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Header.Get("Referer") == "" {
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, "<html><body><h1><B>You have no authority to access this router!</B></h1></body></html>")
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, `<html><body>var statusPara = new Array(1,1,1,1,1,1,"3.15.9 Build 140724 Rel.63227n","WR841N v8 00000000",0,1);</body></html>`)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != expected {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, `<html><body>dashboard</body></html>`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &wr841nServerForServe{server: srv}
}

// listenLoopback opens a random TCP port on the loopback interface
// so serve tests never bind a fixed port.
func listenLoopback() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}

func runServeWithAdapter(t *testing.T, adapter *tplinkwr841v8.Adapter) (string, func()) {
	t.Helper()
	mux := http.NewServeMux()
	store := &sessionStore{password: []byte("hunter2")}
	registerRoutes(mux, adapter, store)

	ln, err := listenLoopback()
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	return "http://" + ln.Addr().String(), func() { _ = srv.Close() }
}

func TestServe_Healthz(t *testing.T) {
	srv := newWR841NForServe(t, "admin", "hunter2")
	host := strings.TrimPrefix(srv.server.URL, "http://")
	a := tplinkwr841v8.New(host, transport.WithTimeout(2*time.Second))
	if err := a.Login(context.Background(), "admin", "hunter2"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	addr, teardown := runServeWithAdapter(t, a)
	defer teardown()
	resp, err := http.Get(addr + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status: %d want 200", resp.StatusCode)
	}
}

func TestServe_Device(t *testing.T) {
	srv := newWR841NForServe(t, "admin", "hunter2")
	host := strings.TrimPrefix(srv.server.URL, "http://")
	a := tplinkwr841v8.New(host, transport.WithTimeout(2*time.Second))
	if err := a.Login(context.Background(), "admin", "hunter2"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	addr, teardown := runServeWithAdapter(t, a)
	defer teardown()
	resp, err := http.Get(addr + "/v0/device")
	if err != nil {
		t.Fatalf("GET /v0/device: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status: %d want 200, body: %s", resp.StatusCode, readAll(t, resp.Body))
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if data, ok := body["data"].(map[string]any); !ok || data["firmwareVersion"] == nil || body["state"] != "verified" {
		t.Errorf("body missing firmwareVersion: %v", body)
	}
}

func TestServe_CapabilitiesReflectVerification(t *testing.T) {
	srv := newWR841NForServe(t, "admin", "hunter2")
	host := strings.TrimPrefix(srv.server.URL, "http://")
	a := tplinkwr841v8.New(host, transport.WithTimeout(2*time.Second))
	if err := a.Login(context.Background(), "admin", "hunter2"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	addr, teardown := runServeWithAdapter(t, a)
	defer teardown()

	resp, err := http.Get(addr + "/v0/capabilities")
	if err != nil {
		t.Fatalf("GET /v0/capabilities: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d want 200", resp.StatusCode)
	}

	var body domain.Observation[map[string]string]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for capability, want := range map[string]string{
		"device":            "verified",
		"status":            "verified",
		"clients":           "verified",
		"wireless_security": "unsupported_or_unverified",
		"wps":               "unsupported_or_unverified",
		"dmz":               "unsupported_or_unverified",
		"upnp":              "unsupported_or_unverified",
		"remote_management": "unsupported_or_unverified",
		"forwarding":        "unsupported_or_unverified",
	} {
		if got := body.Data[capability]; got != want {
			t.Errorf("%s: got %q want %q", capability, got, want)
		}
	}
}

func TestServe_Status(t *testing.T) {
	srv := newWR841NForServe(t, "admin", "hunter2")
	host := strings.TrimPrefix(srv.server.URL, "http://")
	a := tplinkwr841v8.New(host, transport.WithTimeout(2*time.Second))
	if err := a.Login(context.Background(), "admin", "hunter2"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	addr, teardown := runServeWithAdapter(t, a)
	defer teardown()
	resp, err := http.Get(addr + "/v0/status")
	if err != nil {
		t.Fatalf("GET /v0/status: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status: %d want 200, body: %s", resp.StatusCode, readAll(t, resp.Body))
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if data, ok := body["data"].(map[string]any); !ok || data["reachable"] == nil || body["state"] != "verified" {
		t.Errorf("body missing reachable: %v", body)
	}
}

func readAll(t *testing.T, r io.Reader) string {
	t.Helper()
	body, _ := io.ReadAll(r)
	return string(body)
}

func TestServe_SecurityEndpoints(t *testing.T) {
	srv := newWR841NForServe(t, "admin", "hunter2")
	host := strings.TrimPrefix(srv.server.URL, "http://")
	a := tplinkwr841v8.New(host, transport.WithTimeout(2*time.Second))
	if err := a.Login(context.Background(), "admin", "hunter2"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	addr, teardown := runServeWithAdapter(t, a)
	defer teardown()
	for _, p := range []string{
		"/v0/security/wireless",
		"/v0/security/wps",
		"/v0/security/dmz",
		"/v0/security/upnp",
		"/v0/security/remote-management",
		"/v0/security/forwarding",
	} {
		t.Run(p, func(t *testing.T) {
			resp, err := http.Get(addr + p)
			if err != nil {
				t.Fatalf("GET %s: %v", p, err)
			}
			defer resp.Body.Close()
			wantStatus := http.StatusNotFound
			if strings.HasSuffix(p, "/wireless") {
				wantStatus = http.StatusOK
			}
			if resp.StatusCode != wantStatus {
				t.Errorf("status: %d want %d", resp.StatusCode, wantStatus)
			}
			body := readAll(t, resp.Body)
			result, err := domain.DecodeObservation([]byte(body))
			if err != nil || result.State != domain.UnsupportedOrUnverified {
				t.Fatalf("envelope: %s (%v)", body, err)
			}
			if wantStatus == http.StatusOK {
				var wifi map[string]any
				if err := json.Unmarshal(result.Data, &wifi); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(result.Reason, "3.15.9") || result.Provenance != domain.ProvenanceObserved ||
					wifi["securityType"] != "wpa2-psk" || wifi["hasPreSharedKey"] != true || len(wifi) != 7 ||
					strings.Contains(body, "TEST_PSK") {
					t.Errorf("wireless data not safely projected: %s", body)
				}
			}
		})
	}
}

type observationTestAdapter struct {
	domain.RouterAdapter
	clients  []domain.Client
	security domain.SecurityState
	err      error
}

func (a observationTestAdapter) Identify(context.Context) (domain.DeviceInfo, error) {
	return domain.DeviceInfo{Vendor: "configured adapter", Provenance: domain.ProvenanceObserved}, a.err
}

func (a observationTestAdapter) Status(context.Context) (domain.RouterStatus, error) {
	return domain.RouterStatus{}, a.err
}

func (a observationTestAdapter) Clients(context.Context) ([]domain.Client, error) {
	return a.clients, a.err
}

func (a observationTestAdapter) SecurityCapability(context.Context, string) (domain.SecurityState, error) {
	return a.security, a.err
}

func TestObservationHandler(t *testing.T) {
	security := domain.SecurityState{
		WPSEnabled: domain.False, DMZEnabled: domain.False, ActiveUPnPMappings: domain.SomeInt(0),
		RemoteManagementEnabled: domain.False, ForwardingRules: domain.SomeInt(0),
		Provenance: domain.ProvenanceObserved,
	}
	for _, name := range []string{"wps", "dmz", "upnp", "remote-management", "forwarding"} {
		for _, tc := range []struct {
			name    string
			adapter observationTestAdapter
			state   domain.ObservationState
			status  int
		}{
			{"false_or_zero", observationTestAdapter{security: security}, domain.Verified, http.StatusOK},
			{"empty", observationTestAdapter{}, domain.Absent, http.StatusOK},
			{"absent", observationTestAdapter{err: domain.ErrObservationAbsent}, domain.Absent, http.StatusOK},
			{"unsupported", observationTestAdapter{err: domain.ErrUnverifiedEndpoint}, domain.UnsupportedOrUnverified, http.StatusNotFound},
			{"unavailable", observationTestAdapter{err: domain.ErrUnexpectedResponse}, domain.Unavailable, http.StatusServiceUnavailable},
		} {
			t.Run(name+"/"+tc.name, func(t *testing.T) {
				assertObservation(t, observationHandler(securityReader(tc.adapter, name)), tc.state, tc.status)
			})
		}
	}
	t.Run("empty clients", func(t *testing.T) {
		result := assertObservation(t, observationHandler(clientsReader(observationTestAdapter{clients: []domain.Client{}})), domain.Absent, http.StatusOK)
		if string(result.Data) != "[]" || result.Provenance != domain.ProvenanceObserved {
			t.Fatalf("unexpected empty lease evidence: %+v", result)
		}
	})
	t.Run("empty device", func(t *testing.T) {
		assertObservation(t, observationHandler(deviceReader(observationTestAdapter{})), domain.Unavailable, http.StatusServiceUnavailable)
	})
	t.Run("empty status", func(t *testing.T) {
		assertObservation(t, observationHandler(statusReader(observationTestAdapter{})), domain.Absent, http.StatusOK)
	})
}

func assertObservation(t *testing.T, handler http.HandlerFunc, want domain.ObservationState, status int) domain.Observation[json.RawMessage] {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	got, err := domain.DecodeObservation(w.Body.Bytes())
	if err != nil || w.Code != status || got.State != want {
		t.Fatalf("HTTP %d %s (%v), want HTTP %d %s", w.Code, w.Body.String(), err, status, want)
	}
	return got
}

func TestServe_RejectsNonLoopbackAddr(t *testing.T) {
	if isLoopbackAddr("0.0.0.0:8484") {
		t.Errorf("0.0.0.0 should not be accepted as loopback")
	}
	if isLoopbackAddr("192.168.1.1:8484") {
		t.Errorf("RFC1918 should not be accepted as loopback")
	}
	if !isLoopbackAddr("127.0.0.1:8484") {
		t.Errorf("127.0.0.1 must be accepted as loopback")
	}
	if !isLoopbackAddr("localhost:8484") {
		t.Errorf("localhost must be accepted as loopback")
	}
}
