// Package sercomm implements the domain.RouterAdapter contract for Sercomm
// and Sagemcom router firmware (such as the AT904X).
//
// Protocol verification (2026-09-05):
//   - Identification & WAN parameters: unauthenticated GET /data/user_lang.json
//   - Login: POST /data/login.json with HMAC-SHA256 challenge-response
//   - Read-only RPC observation: POST /data/data.cgi with JSON-RPC 2.0 (method "GET")
//
// Every operation is read-only.
package sercomm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Quiarom/router-core/internal/domain"
	"github.com/Quiarom/router-core/internal/transport"
)

const (
	maxBodySize    = 2 << 20 // 2 MiB read cap
	defaultTimeout = 5 * time.Second
)

type session struct {
	authenticated bool
	username      string
	encryptionKey string
	salt          string
}

// Adapter implements domain.RouterAdapter for Sercomm devices.
type Adapter struct {
	host       string
	client     *http.Client
	timeout    time.Duration
	mu         sync.Mutex
	session    *session
	latestLang map[string]string
}

// New creates a new Sercomm router adapter for the specified host.
func New(host string, opts ...transport.Option) *Adapter {
	cleanHost := strings.TrimRight(host, "/")
	if !strings.HasPrefix(cleanHost, "http://") && !strings.HasPrefix(cleanHost, "https://") {
		cleanHost = "http://" + cleanHost
	}

	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: defaultTimeout,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			if req.Response == nil || req.Response.Request == nil {
				return errors.New("router-core: redirect missing source request")
			}
			if !strings.EqualFold(req.URL.Host, req.Response.Request.URL.Host) {
				return errors.New("router-core: cross-host redirect refused")
			}
			return nil
		},
	}

	return &Adapter{
		host:    cleanHost,
		client:  client,
		timeout: defaultTimeout,
	}
}

// Detect checks if the target host exposes the Sercomm /data/user_lang.json endpoint.
func Detect(ctx context.Context, host string) bool {
	cleanHost := strings.TrimRight(host, "/")
	if !strings.HasPrefix(cleanHost, "http://") && !strings.HasPrefix(cleanHost, "https://") {
		cleanHost = "http://" + cleanHost
	}
	targetURL := fmt.Sprintf("%s/data/user_lang.json?_=%d", cleanHost, time.Now().UnixMilli())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "router-core")

	client := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16384))
	if err != nil {
		return false
	}

	lang, err := ParseUserLang(body)
	if err != nil {
		return false
	}
	_, hasFW := lang["fw_version"]
	_, hasSalt := lang["salt"]
	return hasFW || hasSalt
}

func (a *Adapter) dispatch(ctx context.Context, method, path string, body []byte, contentType string) ([]byte, int, error) {
	u, err := url.Parse(a.host + path)
	if err != nil {
		return nil, 0, fmt.Errorf("router-core: invalid URL %q: %w", a.host+path, err)
	}

	hostOnly := u.Host
	if h, _, err := net.SplitHostPort(u.Host); err == nil {
		hostOnly = h
	}
	if !transport.IsAllowedHost(hostOnly) {
		return nil, 0, fmt.Errorf("router-core: host %q is not an allowed RFC1918/loopback address", u.Host)
	}

	reqCtx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(reqCtx, method, u.String(), reader)
	if err != nil {
		return nil, 0, fmt.Errorf("router-core: create request: %w", err)
	}

	req.Header.Set("User-Agent", "router-core")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", a.host+"/pages.html")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", domain.ErrUnreachable, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize+1))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("%w: read response: %v", domain.ErrUnreachable, err)
	}
	if len(respBody) > maxBodySize {
		return nil, resp.StatusCode, errors.New("router-core: response exceeds 2 MiB cap")
	}

	return respBody, resp.StatusCode, nil
}

func (a *Adapter) fetchUserLang(ctx context.Context) (map[string]string, error) {
	path := fmt.Sprintf("/data/user_lang.json?_=%d", time.Now().UnixMilli())
	body, status, err := a.dispatch(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d from %s", domain.ErrUnexpectedResponse, status, path)
	}
	lang, err := ParseUserLang(body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrUnexpectedResponse, err)
	}
	a.mu.Lock()
	a.latestLang = lang
	a.mu.Unlock()
	return lang, nil
}

// Login authenticates against Sercomm using HMAC-SHA256 challenge-response.
func (a *Adapter) Login(ctx context.Context, username, password string) error {
	lang, err := a.fetchUserLang(ctx)
	if err != nil {
		return err
	}

	encryptionKey := lang["encryption_key"]
	salt := lang["salt"]
	if encryptionKey == "" || salt == "" {
		return fmt.Errorf("%w: missing encryption parameters in user_lang.json", domain.ErrUnexpectedResponse)
	}

	hashedPassword := ComputePassword(password, encryptionKey)
	loginForm := url.Values{}
	loginForm.Set("LoginName", username)
	loginForm.Set("LoginPWD", hashedPassword)

	path := fmt.Sprintf("/data/login.json?_=%d", time.Now().UnixMilli())
	respBody, status, err := a.dispatch(ctx, http.MethodPost, path, []byte(loginForm.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("%w: HTTP %d on login", domain.ErrUnreachable, status)
	}

	respStr := strings.TrimSpace(string(respBody))
	// Sercomm returns '"1"' or '[ ]' on success
	if strings.HasPrefix(respStr, `"1"`) || strings.HasPrefix(respStr, `[ ]`) || strings.HasPrefix(respStr, `[{"`) {
		a.mu.Lock()
		a.session = &session{
			authenticated: true,
			username:      username,
			encryptionKey: encryptionKey,
			salt:          salt,
		}
		a.mu.Unlock()
		return nil
	}

	// Status codes: "2" = already logged in, "3"/"4" = bad password
	if strings.HasPrefix(respStr, `"2"`) {
		return domain.ErrSessionConflict
	}
	if strings.HasPrefix(respStr, `"3"`) || strings.HasPrefix(respStr, `"4"`) {
		return domain.ErrUnauthenticated
	}

	return fmt.Errorf("%w: unrecognized login response %q", domain.ErrUnexpectedResponse, respStr)
}

// Identify extracts device info from /data/user_lang.json.
func (a *Adapter) Identify(ctx context.Context) (domain.DeviceInfo, error) {
	lang, err := a.fetchUserLang(ctx)
	if err != nil {
		return domain.DeviceInfo{}, err
	}

	fw := lang["fw_version"]
	model := "AT904X"
	if strings.Contains(fw, "-") {
		parts := strings.SplitN(fw, "-", 2)
		if parts[0] != "" {
			model = parts[0]
		}
	}

	a.mu.Lock()
	auth := domain.Unknown
	if a.session != nil && a.session.authenticated {
		auth = domain.True
	}
	a.mu.Unlock()

	return domain.DeviceInfo{
		Vendor:            "Sercomm",
		Model:             model,
		HardwareVersion:   domain.NewUntrusted(model, "router:sercomm"),
		FirmwareVersion:   domain.NewUntrusted(fw, "router:sercomm"),
		ManagementAddress: a.host,
		Authenticated:     auth,
		Provenance:        domain.ProvenanceObserved,
	}, nil
}

// Status extracts operational WAN status and reachability.
func (a *Adapter) Status(ctx context.Context) (domain.RouterStatus, error) {
	lang, err := a.fetchUserLang(ctx)
	if err != nil {
		return domain.RouterStatus{}, err
	}

	wanStatus := domain.WANDisconnected
	wanIP := lang["wan_ip4_addr"]
	if wanIP != "" && wanIP != "0.0.0.0" {
		wanStatus = domain.WANConnected
	}

	return domain.RouterStatus{
		Reachable:  domain.True,
		WANStatus:  wanStatus,
		Provenance: domain.ProvenanceObserved,
	}, nil
}

// Clients queries /data/data.cgi with JSON-RPC "hosts.@host" to list LAN devices.
func (a *Adapter) Clients(ctx context.Context) ([]domain.Client, error) {
	a.mu.Lock()
	sess := a.session
	a.mu.Unlock()
	if sess == nil || !sess.authenticated {
		return nil, domain.ErrObservationAbsent
	}

	rpcReq := []RPCRequest{
		{
			JSONRPC: "2.0",
			Method:  "GET",
			Params:  "hosts.@host",
			ID:      1,
		},
	}
	payload, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("marshal rpc request: %w", err)
	}

	path := fmt.Sprintf("/data/data.cgi?_=%d", time.Now().UnixMilli())
	respBody, status, err := a.dispatch(ctx, http.MethodPost, path, payload, "application/json")
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d from data.cgi", domain.ErrUnreachable, status)
	}

	var rpcResponses []RPCResponse
	if err := json.Unmarshal(respBody, &rpcResponses); err != nil {
		return nil, fmt.Errorf("%w: unmarshal rpc response: %v", domain.ErrUnexpectedResponse, err)
	}

	var rawResult json.RawMessage
	for _, r := range rpcResponses {
		if r.ID == 1 {
			if r.Error != nil {
				return nil, fmt.Errorf("sercomm rpc error (%d): %s", r.Error.Code, r.Error.Message)
			}
			rawResult = r.Result
			break
		}
	}
	if rawResult == nil {
		return []domain.Client{}, nil
	}

	hostItems, err := ParseHostsResult(rawResult)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrUnexpectedResponse, err)
	}

	clients := make([]domain.Client, 0, len(hostItems))
	for _, item := range hostItems {
		// Only list active/alive clients, or include all if alive field is not '0'
		if item.Alive == "0" {
			continue
		}
		name := item.Hostname
		if name == "" {
			name = "Dispositivo sin nombre"
		}
		clients = append(clients, domain.Client{
			Name:       domain.NewUntrusted(name, "router:sercomm"),
			IP:         item.HostIP,
			MAC:        item.HostMAC,
			Provenance: domain.ProvenanceObserved,
		})
	}

	return clients, nil
}

// Security queries /data/data.cgi with JSON-RPC "wireless.general" for WPS state.
func (a *Adapter) Security(ctx context.Context) (domain.SecurityState, error) {
	state := domain.SecurityState{
		WPSEnabled:              domain.Unknown,
		DMZEnabled:              domain.Unknown,
		UPnPEnabled:             domain.Unknown,
		RemoteManagementEnabled: domain.Unknown,
		Provenance:              domain.ProvenanceObserved,
	}
	state.MarkUnsupported("dmz", "endpoint not implemented on Sercomm general view")
	state.MarkUnsupported("upnp", "endpoint not implemented on Sercomm general view")
	state.MarkUnsupported("remote-management", "endpoint not implemented on Sercomm general view")

	a.mu.Lock()
	sess := a.session
	a.mu.Unlock()
	if sess == nil || !sess.authenticated {
		return state, nil
	}

	rpcReq := []RPCRequest{
		{
			JSONRPC: "2.0",
			Method:  "GET",
			Params:  "wireless.general",
			ID:      1,
		},
	}
	payload, err := json.Marshal(rpcReq)
	if err != nil {
		return state, fmt.Errorf("marshal rpc request: %w", err)
	}

	path := fmt.Sprintf("/data/data.cgi?_=%d", time.Now().UnixMilli())
	respBody, status, err := a.dispatch(ctx, http.MethodPost, path, payload, "application/json")
	if err != nil {
		return state, err
	}
	if status != http.StatusOK {
		return state, fmt.Errorf("%w: HTTP %d from data.cgi", domain.ErrUnreachable, status)
	}

	var rpcResponses []RPCResponse
	if err := json.Unmarshal(respBody, &rpcResponses); err != nil {
		return state, fmt.Errorf("%w: unmarshal rpc response: %v", domain.ErrUnexpectedResponse, err)
	}

	for _, r := range rpcResponses {
		if r.ID == 1 && r.Result != nil {
			gen, err := ParseWirelessResult(r.Result)
			if err == nil {
				if gen.WPS == "1" {
					state.WPSEnabled = domain.True
				} else if gen.WPS == "0" {
					state.WPSEnabled = domain.False
				}
			}
			break
		}
	}

	return state, nil
}

// SecurityCapability implements the optional securityCapable contract for serve.
func (a *Adapter) SecurityCapability(ctx context.Context, name string) (domain.SecurityState, error) {
	switch name {
	case "wireless_security", "wireless", "wps", "wps_state":
		return a.Security(ctx)
	default:
		return domain.SecurityState{}, domain.ErrUnverifiedEndpoint
	}
}
