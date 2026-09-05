package sercomm_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Quiarom/router-core/internal/adapters/sercomm"
	"github.com/Quiarom/router-core/internal/domain"
)

func TestComputePassword(t *testing.T) {
	// Vector tested against sha.js reference implementation with key "$1$SERCOMM$"
	res := sercomm.ComputePassword("admin", "0491bfc3288ce42d")
	want := "40602f48d27548b6b7e7c2f0c8eb2af3ad8be90b9674254d7f5b184b9dfd068a"
	if res != want {
		t.Fatalf("got %s, want %s", res, want)
	}
}

func mockSercommServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasPrefix(path, "/data/user_lang.json"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[
				{"lang_code": ""},
				{"fw_version": "AT904X-03.01s"},
				{"wan_ip4_addr": "217.34.98.17s"},
				{"encryption_key": "0491bfc3288ce42d"},
				{"salt": "0491BFC3288CE42D"}
			]`))
		case strings.HasPrefix(path, "/data/login.json"):
			_ = r.ParseForm()
			pwd := r.Form.Get("LoginPWD")
			if pwd == "40602f48d27548b6b7e7c2f0c8eb2af3ad8be90b9674254d7f5b184b9dfd068a" {
				w.Header().Set("Set-Cookie", "session_id=mock123; path=/")
				_, _ = w.Write([]byte(`"1"`))
			} else {
				_, _ = w.Write([]byte(`"3"`))
			}
		case strings.HasPrefix(path, "/data/data.cgi"):
			cookie, err := r.Cookie("session_id")
			if err != nil || cookie.Value != "mock123" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[
				{
					"id": 1,
					"result": {
						"hosts.@host": [
							{
								"hostname": "test-phone",
								"hostip": "192.168.50.15",
								"hostmac": "aa:bb:cc:dd:ee:ff",
								"conntype": "1",
								"alive": "1"
							},
							{
								"hostname": "offline-pc",
								"hostip": "192.168.50.20",
								"hostmac": "11:22:33:44:55:66",
								"conntype": "0",
								"alive": "0"
							}
						],
						"wireless.general": {
							"wps": "0",
							"schedule": "0"
						}
					}
				}
			]`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestAdapter_Identify(t *testing.T) {
	ts := mockSercommServer(t)
	defer ts.Close()

	adapter := sercomm.New(ts.URL)
	info, err := adapter.Identify(context.Background())
	if err != nil {
		t.Fatalf("Identify failed: %v", err)
	}

	if info.Vendor != "Sercomm" {
		t.Errorf("Vendor: got %s, want Sercomm", info.Vendor)
	}
	if info.Model != "AT904X" {
		t.Errorf("Model: got %s, want AT904X", info.Model)
	}
	if info.FirmwareVersion.Value() != "AT904X-03.01" {
		t.Errorf("FirmwareVersion: got %s, want AT904X-03.01", info.FirmwareVersion.Value())
	}
	if info.Authenticated != domain.Unknown {
		t.Errorf("Authenticated before login: got %v, want Unknown", info.Authenticated)
	}
}

func TestAdapter_Status(t *testing.T) {
	ts := mockSercommServer(t)
	defer ts.Close()

	adapter := sercomm.New(ts.URL)
	status, err := adapter.Status(context.Background())
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	if status.Reachable != domain.True {
		t.Errorf("Reachable: got %v, want True", status.Reachable)
	}
	if status.WANStatus != domain.WANConnected {
		t.Errorf("WANStatus: got %v, want Connected", status.WANStatus)
	}
}

func TestAdapter_LoginAndClients(t *testing.T) {
	ts := mockSercommServer(t)
	defer ts.Close()

	adapter := sercomm.New(ts.URL)

	// Attempting clients before login reports absent
	_, err := adapter.Clients(context.Background())
	if !errors.Is(err, domain.ErrObservationAbsent) {
		t.Fatalf("Clients before login should return ErrObservationAbsent, got %v", err)
	}

	// Login with invalid password fails
	err = adapter.Login(context.Background(), "admin", "wrongpass")
	if err == nil {
		t.Fatal("Login with wrong password should fail")
	}

	// Login with valid password succeeds
	err = adapter.Login(context.Background(), "admin", "admin")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	// Clients now returns active clients
	clients, err := adapter.Clients(context.Background())
	if err != nil {
		t.Fatalf("Clients failed: %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("expected 1 active client, got %d", len(clients))
	}
	if clients[0].Name.Value() != "test-phone" {
		t.Errorf("client name: got %s, want test-phone", clients[0].Name.Value())
	}
	if clients[0].IP != "192.168.50.15" {
		t.Errorf("client IP: got %s, want 192.168.50.15", clients[0].IP)
	}

	// Security returns WPS state
	sec, err := adapter.Security(context.Background())
	if err != nil {
		t.Fatalf("Security failed: %v", err)
	}
	if sec.WPSEnabled != domain.False {
		t.Errorf("WPSEnabled: got %v, want False", sec.WPSEnabled)
	}
}

func TestDetect(t *testing.T) {
	ts := mockSercommServer(t)
	defer ts.Close()

	if !sercomm.Detect(context.Background(), ts.URL) {
		t.Errorf("Detect should return true for mock Sercomm server")
	}
}
