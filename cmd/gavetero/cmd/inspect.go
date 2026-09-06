// Package cmd: the inspect subcommand. It runs router-core
// as a sidecar and queries its HTTP API, formatting the result
// for human or machine consumption.
//
// The default mock mode uses the embedded fixture (TL-WR841N/ND
// v8.4 on firmware 3.15.9) and does not touch the network.
// With --live --host X, the sidecar talks to a real router at
// X (default port 80). The same HTTP API exposes the router's
// observations to gavetero inspect.
//
// The output uses the four-state vocabulary the API surface
// uses: verified, absent, unsupported_or_unverified, unavailable.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Quiarom/router-core/cmd/gavetero/cmd/sidecars"
	"github.com/spf13/cobra"
)

type inspectOptions struct {
	Output              string
	Live                bool
	Host                string
	RouterUser          string
	RouterPasswordStdin bool
}

func newInspectCmd() *cobra.Command {
	var (
		output              string
		live                bool
		host                string
		routerUser          string
		routerPasswordStdin bool
	)

	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Show the current router observations",
		Long: `Run router-core as a sidecar and query its HTTP API.

By default, the sidecar runs in mock mode using the embedded
fixture (TP-Link TL-WR841N/ND v8.4 on firmware 3.15.9). No
network access required.

With --live --host X, the sidecar talks to a real router at
address X (default port 80). Use --router-user and
--router-password-stdin to provide credentials (admin/admin
is the default for TP-Link WR841N; other routers may differ).

The output uses the four-state vocabulary the API surface
uses: verified, absent, unsupported_or_unverified, unavailable.`,
		Example: `  gvt inspect
  gvt inspect --live --host 192.168.0.1
  gvt inspect --live --host 192.168.0.1 --router-password-stdin
  gvt inspect --output json
  gvt inspect --output jsonl`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInspect(cmd.OutOrStdout(), cmd.ErrOrStderr(), inspectOptions{
				Output:              output,
				Live:                live,
				Host:                host,
				RouterUser:          routerUser,
				RouterPasswordStdin: routerPasswordStdin,
			})
		},
	}
	cmd.Flags().StringVar(&output, "output", "human",
		"output format: human, json, jsonl")
	cmd.Flags().BoolVar(&live, "live", false,
		"connect to a real router (default uses the embedded mock fixture)")
	cmd.Flags().StringVar(&host, "host", "",
		"router address for --live mode (e.g. 192.168.0.1)")
	cmd.Flags().StringVar(&routerUser, "router-user", "admin",
		"username for router admin login (used with --live)")
	cmd.Flags().BoolVar(&routerPasswordStdin, "router-password-stdin", false,
		"read the router admin password from stdin (used with --live)")
	return cmd
}

func runInspect(stdout, stderr io.Writer, opts inspectOptions) error {
	bin, err := findRouterCoreBin()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Decide the sidecar mode. Live mode: the sidecar binds to a
	// loopback port and proxies to the real router at opts.Host.
	// Mock mode (default): the sidecar uses the embedded fixture.
	var sidecarArgs []string
	var routerURL string
	if opts.Live {
		if opts.Host == "" {
			return fmt.Errorf("--live requires --host X (router address)")
		}
		addr, lerr := reserveLoopbackAddr()
		if lerr != nil {
			return fmt.Errorf("reserve loopback port: %w", lerr)
		}
		sidecarArgs = []string{"serve", "--host", opts.Host, "--addr", addr}
		if opts.RouterUser != "" {
			sidecarArgs = append(sidecarArgs, "--username", opts.RouterUser)
		}
		if opts.RouterPasswordStdin {
			sidecarArgs = append(sidecarArgs, "--password-stdin")
		}
		routerURL = "http://" + addr
	} else {
		addr, lerr := reserveLoopbackAddr()
		if lerr != nil {
			return fmt.Errorf("reserve loopback port: %w", lerr)
		}
		sidecarArgs = []string{"serve", "--mock", "--addr", addr}
		routerURL = "http://" + addr
	}

	sidecar := exec.CommandContext(ctx, bin, sidecarArgs...)
	sidecar.Stdout = io.Discard
	sidecar.Stderr = stderr
	if err := sidecar.Start(); err != nil {
		return fmt.Errorf("start router-core: %w", err)
	}
	defer func() {
		if sidecar.Process != nil {
			_ = sidecar.Process.Kill()
		}
		_ = sidecar.Wait()
	}()

	// Wait for /v0/capabilities.
	routerClient := &http.Client{Timeout: 2 * time.Second}
	if err := waitForReady(ctx, routerClient, routerURL+"/v0/capabilities"); err != nil {
		return fmt.Errorf("router-core never became ready at %s: %w", routerURL, err)
	}

	caps := map[string]string{}
	if body, err := getJSON(ctx, routerClient, routerURL+"/v0/capabilities"); err == nil {
		var wrapper struct {
			Capabilities map[string]string `json:"capabilities"`
		}
		_ = json.Unmarshal(body, &wrapper)
		caps = wrapper.Capabilities
	}
	device := map[string]interface{}{}
	if body, err := getJSON(ctx, routerClient, routerURL+"/v0/device"); err == nil {
		_ = json.Unmarshal(body, &device)
	}
	status := map[string]interface{}{}
	if body, err := getJSON(ctx, routerClient, routerURL+"/v0/status"); err == nil {
		_ = json.Unmarshal(body, &status)
	}
	clients := []map[string]interface{}{}
	if body, err := getJSON(ctx, routerClient, routerURL+"/v0/clients"); err == nil {
		var wrapper struct {
			Clients []map[string]interface{} `json:"clients"`
		}
		_ = json.Unmarshal(body, &wrapper)
		clients = wrapper.Clients
	}

	switch opts.Output {
	case "json":
		return writeJSON(stdout, map[string]any{
			"router":       inspectSource(opts),
			"mode":         inspectMode(opts),
			"capabilities": caps,
			"device":       device,
			"status":       status,
			"clients":      clients,
		})
	case "jsonl":
		return renderAskJSONL(stdout, map[string]any{
			"kind":     "inspect",
			"source":   inspectSource(opts),
			"mode":     inspectMode(opts),
			"router":   device,
			"status":   status,
			"clients":  clients,
			"caps":     caps,
		})
	default:
		return renderInspectHuman(stdout, caps, device, status, clients, opts)
	}
}

func inspectSource(opts inspectOptions) string {
	if opts.Live {
		return "live router at " + opts.Host
	}
	return "fixture (mock mode)"
}

func inspectMode(opts inspectOptions) string {
	if opts.Live {
		return "live"
	}
	return "mock"
}

func renderInspectHuman(stdout io.Writer, caps map[string]string, device, status map[string]interface{}, clients []map[string]interface{}, opts inspectOptions) error {
	modeLabel := "mock (fixture-backed, no network)"
	if opts.Live {
		modeLabel = "live (router at " + opts.Host + ")"
	}
	fmt.Fprintln(stdout, "Gavetero Inspect")
	fmt.Fprintln(stdout, "================")
	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "Mode:  %s\n", modeLabel)
	fmt.Fprintln(stdout)
	if len(device) > 0 {
		fmt.Fprintln(stdout, "Device")
		fmt.Fprintln(stdout, "------")
		for _, k := range []string{"vendor", "model", "hardwareVersion", "firmwareVersion", "managementAddress", "authenticated", "provenance"} {
			if v, ok := device[k]; ok {
				fmt.Fprintf(stdout, "  %-18s %v\n", k, v)
			}
		}
		fmt.Fprintln(stdout)
	}
	if len(status) > 0 {
		fmt.Fprintln(stdout, "Status")
		fmt.Fprintln(stdout, "------")
		for _, k := range []string{"reachable", "wanStatus", "uptimeSeconds", "provenance"} {
			if v, ok := status[k]; ok {
				fmt.Fprintf(stdout, "  %-18s %v\n", k, v)
			}
		}
		fmt.Fprintln(stdout)
	}
	if len(caps) > 0 {
		fmt.Fprintln(stdout, "Capabilities")
		fmt.Fprintln(stdout, "------------")
		order := []string{
			"device", "status", "clients",
			"wireless", "wireless_security",
			"wps", "wps_state",
			"dmz", "dmz_state",
			"upnp", "upnp_state",
			"remote_management",
			"forwarding", "forwarding_rules",
		}
		seen := map[string]bool{}
		for _, k := range order {
			if v, ok := caps[k]; ok {
				fmt.Fprintf(stdout, "  %-22s %s\n", k, v)
				seen[k] = true
			}
		}
		extras := []string{}
		for k := range caps {
			if !seen[k] {
				extras = append(extras, k)
			}
		}
		if len(extras) > 0 {
			for _, k := range extras {
				fmt.Fprintf(stdout, "  %-22s %s\n", k, caps[k])
			}
		}
		fmt.Fprintln(stdout)
	}
	if clients != nil {
		fmt.Fprintf(stdout, "Clients (%d observed)\n", len(clients))
		fmt.Fprintln(stdout, "---------------------")
		for _, c := range clients {
			ip := toString(flattenMapValue(c, "ip"))
			mac := toString(flattenMapValue(c, "mac"))
			name := toString(flattenMapValue(c, "name"))
			fmt.Fprintf(stdout, "  %-18s %-20s %s\n", ip, mac, name)
		}
		fmt.Fprintln(stdout)
	}
	fmt.Fprintln(stdout, "Legend:")
	fmt.Fprintln(stdout, "  verified                     adapter read the value")
	fmt.Fprintln(stdout, "  absent                       firmware does not implement")
	fmt.Fprintln(stdout, "  unsupported_or_unverified    runtime has no parser")
	fmt.Fprintln(stdout, "  unavailable                  transport failure")
	return nil
}

// flattenMapValue returns the inner string of an Untrusted
// (or any) map, or the empty string if absent.
func flattenMapValue(m map[string]interface{}, key string) interface{} {
	if m == nil {
		return nil
	}
	if v, ok := m[key]; ok {
		return flattenValueInspect(v)
	}
	return nil
}

func flattenValueInspect(v interface{}) interface{} {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case map[string]interface{}:
		if val, ok := x["value"]; ok {
			if trust, ok2 := x["trust"].(string); ok2 && trust == "untrusted" {
				s := toString(val)
				if s == "" {
					return "(empty)"
				}
				return "~ " + s
			}
			return val
		}
		return x
	}
	return v
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func waitForReady(ctx context.Context, client *http.Client, url string) error {
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode/100 == 2 {
				return nil
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	return fmt.Errorf("timeout")
}

func getJSON(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 2<<20))
}

func findRouterCoreBin() (string, error) {
	// Prefer the embedded sidecar (extracted from the gavetero
	// binary itself). Falls back to the legacy disk search.
	if path, err := sidecars.Get("router-core"); err == nil {
		return path, nil
	}
	if env := os.Getenv("ROUTER_CORE_BIN"); env != "" {
		if _, err := os.Stat(env); err == nil {
			return env, nil
		}
	}
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "router-core")
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate, nil
		}
	}
	if path, err := exec.LookPath("router-core"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf(`router-core sidecar not found.

Gavetero normally embeds router-core inside itself. If you see
this error, the gavetero binary was built without the embed
step. Run from the repo root:

  make build
  make install-user`)
}

func reserveLoopbackAddr() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer ln.Close()
	return ln.Addr().String(), nil
}

func writeJSON(w io.Writer, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}
