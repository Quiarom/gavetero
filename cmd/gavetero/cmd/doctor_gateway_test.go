package cmd

// Regression test for the doctor gateway check: it must report
// the gateway of the OS default route, never the local address
// of an unrelated (e.g. VPN) interface.

import (
	"os"
	"path/filepath"
	"testing"
)

func writeRouteFixture(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "route")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return p
}

func TestDefaultGatewayFromProcRoute(t *testing.T) {
	// Mirrors a host with a LAN default route plus a WireGuard
	// interface holding a local 10.0.0.2/24 address. The old
	// implementation reported the local address; the gateway is
	// 192.168.1.1.
	p := writeRouteFixture(t, "Iface\tDestination\tGateway \tFlags\tRefCnt\tUse\tMetric\tMask\t\tMTU\tWindow\tIRTT\n"+
		"enp34s0\t00000000\t0101A8C0\t0003\t0\t0\t100\t00000000\t0\t0\t0\n"+
		"cliente1\t00000A0A\t00000000\t0001\t0\t0\t50\t00FFFFFF\t0\t0\t0\n"+
		"enp34s0\t0001A8C0\t00000000\t0001\t0\t0\t100\t00FFFFFF\t0\t0\t0\n")
	gw, err := defaultGatewayFromProcRoute(p)
	if err != nil {
		t.Fatalf("defaultGatewayFromProcRoute: %v", err)
	}
	if gw != "192.168.1.1" {
		t.Fatalf("got gateway %q, want %q", gw, "192.168.1.1")
	}
}

func TestDefaultGatewayPrefersLowestMetric(t *testing.T) {
	p := writeRouteFixture(t, "Iface\tDestination\tGateway \tFlags\tRefCnt\tUse\tMetric\tMask\t\tMTU\tWindow\tIRTT\n"+
		"eth0\t00000000\t0201A8C0\t0003\t0\t0\t200\t00000000\t0\t0\t0\n"+
		"eth1\t00000000\t0101A8C0\t0003\t0\t0\t100\t00000000\t0\t0\t0\n")
	gw, err := defaultGatewayFromProcRoute(p)
	if err != nil {
		t.Fatalf("defaultGatewayFromProcRoute: %v", err)
	}
	if gw != "192.168.1.1" {
		t.Fatalf("got gateway %q, want %q", gw, "192.168.1.1")
	}
}

func TestDefaultGatewayNoDefaultRoute(t *testing.T) {
	p := writeRouteFixture(t, "Iface\tDestination\tGateway \tFlags\tRefCnt\tUse\tMetric\tMask\t\tMTU\tWindow\tIRTT\n"+
		"lo\t0000007F\t00000000\t0001\t0\t0\t0\t000000FF\t0\t0\t0\n")
	if _, err := defaultGatewayFromProcRoute(p); err == nil {
		t.Fatal("expected error with no default route, got nil")
	}
}
