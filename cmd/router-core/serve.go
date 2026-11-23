// Package main contains the runtime binary router-core serve, which
// authenticates against a TP-Link WR841N v8.4 and exposes a typed
// read-only HTTP API on the loopback interface. Capabilities the
// firmware does not implement (WPS, UPnP, Remote Management on
// v8.4) return 404 with a structured payload.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"golang.org/x/term"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Quiarom/gavetero/internal/adapters/fixture"
	"github.com/Quiarom/gavetero/internal/adapters/tplinkwr841v8"
	"github.com/Quiarom/gavetero/internal/domain"
	"github.com/Quiarom/gavetero/internal/transport"
)

type sessionStore struct {
	mu       sync.Mutex
	password []byte
}

func (s *sessionStore) get() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.password...)
}

func (s *sessionStore) set(p []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.password {
		s.password[i] = 0
	}
	s.password = append([]byte(nil), p...)
}

// readRouterPassword acquires the router admin password in a way that
// respects the four ways the user might supply it:
//
//  1. --password-stdin flag: read plaintext from os.Stdin (intended for
//     systemd, container secrets, scripts). Refuses if no TTY.
//  2. Interactive TTY: prompt with prompt printed to stderr, then
//     read using golang.org/x/term.ReadPassword which DISABLES
//     terminal echo for the duration of the read. The previous
//     implementation used os.Stdin.Read which did NOT disable
//     echo and so the password was visible on the terminal.
//  3. Non-interactive without --password-stdin: fail fast with
//     an actionable error. The previous 30-second wait was
//     blocking CI and scripts in non-interactive mode.
//
// NEVER accepts --password <secret>: that would put the secret
// on the process command line, where it is visible to ps(1)
// and any /proc snapshot.
func readRouterPassword(stdinFlag bool) ([]byte, error) {
	if stdinFlag {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			return nil, errors.New("--password-stdin is set but stdin is a TTY; refusing to read a password from a terminal in plaintext")
		}
		buf, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		return []byte(strings.TrimRight(string(buf), "\r\n")), nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, errors.New("router-core: no TTY available for interactive password entry. Use --password-stdin to read from a pipe (CI, systemd, container secrets).")
	}
	fmt.Fprintln(os.Stderr, "router-core: enter router admin password (input hidden):")
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return nil, fmt.Errorf("read password from TTY: %w", err)
	}
	return pw, nil
}

func runServeCommand(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	host := fs.String("host", "192.168.1.1", "local router address (RFC1918 literal)")
	addr := fs.String("addr", "127.0.0.1:8484", "loopback HTTP listen address")
	timeout := fs.Duration("timeout", 5*time.Second, "per-request timeout to the router")
	mock := fs.Bool("mock", false, "run against a fixture-backed adapter (no network)")
	mockPath := fs.String("mock-fixture", "", "path to a synthetic fixture (default: fixtures/synthetic/tplink-wr841n-v8)")
	passwordStdin := fs.Bool("password-stdin", false, "read the admin password from stdin (refuses if stdin is a TTY)")
	username := fs.String("username", "admin", "admin username for the router login")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) || err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if !isLoopbackAddr(*addr) {
		return fmt.Errorf("refusing to serve on non-loopback address %q (loopback only)", *addr)
	}
	var adapter domain.RouterAdapter
	var store *sessionStore
	if *mock {
		fixturePath := *mockPath
		if fixturePath == "" {
			fixturePath = "fixtures/synthetic/tplink-wr841n-v8"
		}
		f := fixture.New(fixturePath)
		adapter = f
		store = &sessionStore{password: nil}
		fmt.Fprintf(os.Stderr, "router-core serve: mock mode, fixture=%s\n", fixturePath)
	} else {
		if !isRFC1918OrLoopback(*host) {
			return fmt.Errorf("refusing to observe host %q: not loopback/RFC1918", *host)
		}
		fmt.Fprintf(os.Stderr, "router-core serve: reading admin password from stdin (timeout 30s)\n")
		password, err := readRouterPassword(*passwordStdin)
		if err != nil {
			return fmt.Errorf("read password: %w", err)
		}
		if len(password) == 0 {
			return errors.New("empty password")
		}
		defer zeroBytes(&password)

		store := &sessionStore{password: append([]byte(nil), password...)}
		defer zeroBytes(&store.password)

		liveAdapter := tplinkwr841v8.New(*host, transport.WithTimeout(*timeout))
		loginCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := liveAdapter.Login(loginCtx, *username, string(password)); err != nil {
			return fmt.Errorf("login: %w", err)
		}
		adapter = liveAdapter

		fmt.Fprintf(os.Stderr, "router-core serve: authenticated, listening on %s\n", *addr)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, adapter, store)

	srv := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return srv.ListenAndServe()
}

// isLoopbackAddr refuses 0.0.0.0 and public IPs. The router-core
// service is local-only and must never bind on a routable address.
func isLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsUnspecified() && ip.To4()[0] == 127
}

// isRFC1918OrLoopback is duplicated from the probe binary to keep
// the runtime binary self-contained. The transport layer applies
// the same check; this is a defense-in-depth pre-flight.
func isRFC1918OrLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = strings.Trim(addr, "[]")
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 10 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168)
	}
	return false
}

// zeroBytes overwrites every byte in p with zero. Unlike a Go
// string, a []byte is mutable. This is the best we can do in
// Go without an unsafe zeroing library.
func zeroBytes(p *[]byte) {
	if p == nil {
		return
	}
	for i := range *p {
		(*p)[i] = 0
	}
	*p = (*p)[:0]
}

type observationReader func(context.Context) domain.Observation[any]

type securityCapable interface {
	SecurityCapability(ctx context.Context, name string) (domain.SecurityState, error)
}

type wirelessCapable interface {
	FetchWirelessSecurity(context.Context) (tplinkwr841v8.WirelessSecurity, error)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func registerRoutes(mux *http.ServeMux, adapter domain.RouterAdapter, store *sessionStore) {
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"state": "ok", "http_status": 200})
	})
	readers := map[string]observationReader{
		"device":   deviceReader(adapter),
		"status":   statusReader(adapter),
		"clients":  clientsReader(adapter),
		"wireless": wirelessReader(adapter),
	}
	for _, name := range []string{
		tplinkwr841v8.OpWPS, tplinkwr841v8.OpDMZ, tplinkwr841v8.OpUPnP,
		tplinkwr841v8.OpRemoteManagement, tplinkwr841v8.OpForwarding,
	} {
		readers[name] = securityReader(adapter, name)
	}
	for name, read := range readers {
		prefix := "/v0/security/"
		switch name {
		case "device", "status", "clients":
			prefix = "/v0/"
		}
		mux.HandleFunc(prefix+name, observationHandler(read))
	}
	mux.HandleFunc("/v0/capabilities", observationHandler(capabilitiesReader(readers)))
}

func observationHandler(read observationReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeJSON(w, http.StatusMethodNotAllowed, observation(domain.Unavailable, "method not allowed", domain.ProvenanceAbsent, nil))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		result := read(ctx)
		status := http.StatusOK
		switch {
		case result.State == domain.UnsupportedOrUnverified && result.Data == nil:
			status = http.StatusNotFound
		case result.State == domain.Unavailable:
			status = http.StatusServiceUnavailable
		}
		writeJSON(w, status, result)
	}
}

func observation(state domain.ObservationState, reason string, source domain.Provenance, data any) domain.Observation[any] {
	return domain.Observation[any]{State: state, Reason: reason, Provenance: source, Data: data}
}

func observed(reason string, source domain.Provenance, data any) domain.Observation[any] {
	if data == nil {
		return observation(domain.Unavailable, "observation has no data", domain.ProvenanceAbsent, nil)
	}
	if source != domain.ProvenanceObserved && source != domain.ProvenanceFixture {
		return observation(domain.Unavailable, "observation has no recorded source", domain.ProvenanceAbsent, nil)
	}
	return observation(domain.Verified, reason, source, data)
}

func observationError(err error, name string) domain.Observation[any] {
	switch {
	case errors.Is(err, domain.ErrObservationAbsent):
		return observation(domain.Absent, name+" was not present in the source", domain.ProvenanceAbsent, nil)
	case errors.Is(err, domain.ErrUnverifiedEndpoint):
		return observation(domain.UnsupportedOrUnverified, name+" endpoint has no verified recipe", domain.ProvenanceAbsent, nil)
	case errors.Is(err, domain.ErrCaptureMissing):
		return observation(domain.Unavailable, name+" requires an active router session", domain.ProvenanceAbsent, nil)
	default:
		return observation(domain.Unavailable, name+" could not be obtained or interpreted", domain.ProvenanceAbsent, nil)
	}
}

func deviceReader(adapter domain.RouterAdapter) observationReader {
	return func(ctx context.Context) domain.Observation[any] {
		info, err := adapter.Identify(ctx)
		if err != nil {
			return observationError(err, "device identity")
		}
		if info.FirmwareVersion.Empty() && info.HardwareVersion.Empty() && !info.Authenticated.Known() {
			if _, mock := adapter.(*fixture.Adapter); mock {
				return observation(domain.Absent, "device identity was not present in the fixture", info.Provenance, nil)
			}
			return observationError(domain.ErrUnexpectedResponse, "device identity")
		}
		return observed("read device identity", info.Provenance, info)
	}
}

func statusReader(adapter domain.RouterAdapter) observationReader {
	return func(ctx context.Context) domain.Observation[any] {
		status, err := adapter.Status(ctx)
		if err != nil {
			return observationError(err, "router status")
		}
		if !status.Reachable.Known() && !status.UptimeSecs.Valid && (status.WANStatus == "" || status.WANStatus == domain.WANUnknown) {
			source := status.Provenance
			if source == "" {
				source = domain.ProvenanceAbsent
			}
			return observation(domain.Absent, "no router status fields were observed", source, nil)
		}
		return observed("read router status", status.Provenance, status)
	}
}

func clientsReader(adapter domain.RouterAdapter) observationReader {
	return func(ctx context.Context) domain.Observation[any] {
		clients, err := adapter.Clients(ctx)
		if err != nil {
			return observationError(err, "DHCP lease list")
		}
		if len(clients) == 0 {
			if clients == nil {
				return observation(domain.Absent, "no DHCP lease list was observed", domain.ProvenanceAbsent, nil)
			}
			source := domain.ProvenanceObserved
			if _, mock := adapter.(*fixture.Adapter); mock {
				source = domain.ProvenanceFixture
			}
			return observation(domain.Absent, "no DHCP lease rows were observed", source, []domain.Client{})
		}
		return observed("read DHCP lease rows", clients[0].Provenance, clients)
	}
}

func securityReader(adapter domain.RouterAdapter, name string) observationReader {
	return func(ctx context.Context) domain.Observation[any] {
		sc, ok := adapter.(securityCapable)
		if !ok {
			return observation(domain.UnsupportedOrUnverified, name+" endpoint is not supported by this adapter", domain.ProvenanceAbsent, nil)
		}
		state, err := sc.SecurityCapability(ctx, name)
		if err != nil {
			return observationError(err, name)
		}
		var hasData bool
		switch name {
		case tplinkwr841v8.OpWPS:
			hasData = state.WPSEnabled.Known()
		case tplinkwr841v8.OpDMZ:
			hasData = state.DMZEnabled.Known() || state.DMZHost != ""
		case tplinkwr841v8.OpUPnP:
			hasData = state.UPnPEnabled.Known() || state.ActiveUPnPMappings.Valid
		case tplinkwr841v8.OpRemoteManagement:
			hasData = state.RemoteManagementEnabled.Known() || state.RemoteManagementPort.Valid
		case tplinkwr841v8.OpForwarding:
			hasData = state.ForwardingRules.Valid
		}
		if !hasData {
			source := state.Provenance
			if source == "" {
				source = domain.ProvenanceAbsent
			}
			return observation(domain.Absent, "no "+name+" fields were observed", source, nil)
		}
		return observed("read "+name, state.Provenance, state)
	}
}

func wirelessReader(adapter domain.RouterAdapter) observationReader {
	const history = "Wi-Fi is not verified for the alpha; only historical evidence exists on firmware 3.15.9"
	return func(ctx context.Context) domain.Observation[any] {
		reader, ok := adapter.(wirelessCapable)
		if !ok {
			return observation(domain.UnsupportedOrUnverified, history+"; this adapter has no Wi-Fi reader", domain.ProvenanceAbsent, nil)
		}
		wifi, err := reader.FetchWirelessSecurity(ctx)
		if err != nil {
			return observation(domain.UnsupportedOrUnverified, history+"; no current data could be read", domain.ProvenanceAbsent, nil)
		}
		data := struct {
			Enabled         bool             `json:"enabled"`
			SSID            domain.Untrusted `json:"ssid"`
			SecurityTypeRaw int              `json:"securityTypeRaw"`
			SecurityType    string           `json:"securityType"`
			Cipher          string           `json:"cipher"`
			KeyRenewalSecs  int              `json:"keyRenewalSecs"`
			HasPreSharedKey bool             `json:"hasPreSharedKey"`
		}{
			wifi.Enabled, domain.NewUntrusted(wifi.SSID, "router:wireless-security"), wifi.SecurityTypeRaw,
			wifi.SecurityType, wifi.Cipher, wifi.KeyRenewalSecs, wifi.HasPreSharedKey,
		}
		return observation(domain.UnsupportedOrUnverified, history, domain.ProvenanceObserved, data)
	}
}

func capabilitiesReader(readers map[string]observationReader) observationReader {
	return func(ctx context.Context) domain.Observation[any] {
		matrix := make(map[string]domain.ObservationState, len(readers))
		source := domain.ProvenanceAbsent
		for _, name := range []string{"device", "status", "clients", "wireless", "wps", "dmz", "upnp", "remote-management", "forwarding"} {
			probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			result := readers[name](probeCtx)
			cancel()
			matrix[name] = result.State
			if result.Provenance == domain.ProvenanceFixture || source == domain.ProvenanceAbsent {
				source = result.Provenance
			}
		}
		for alias, name := range map[string]string{
			"wireless_security": "wireless", "wps_state": "wps", "dmz_state": "dmz", "upnp_state": "upnp",
			"remote_management": "remote-management", "forwarding_rules": "forwarding",
		} {
			matrix[alias] = matrix[name]
		}
		return observation(domain.Verified, "summarized observation states; entries are independent", source, matrix)
	}
}
