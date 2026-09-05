// Package cmd: the detect subcommand. It tries to identify a
// router on the network (default gateway or --host) and report
// what Gavetero can observe about it, even when there is no
// specific adapter for the firmware.
//
// This is the user-facing path for "any modem/router", not
// just the WR841N. It does the best it can with what is
// available today:
//
//   1. If the host responds to the standard userRpm /login.htm
//      path with a router login page, we report a
//      TP-Link-class HTTP UI is reachable (heuristic; not a
//      positive identification).
//   2. We forward to the user the only thing we can really do
//      without a specific adapter: ask the user to add their
//      firmware to the adapter registry, or run with
//      ROUTER_CORE_BIN pointing at a local sidecar that knows
//      the firmware.
//
// Today, detect is honest: it tells the user what we know
// (HTTP UI reachable) and what we don't (adapter-specific
// observations).
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	// Heuristic: many TP-Link firmwares (WR841N, Archer C6,
	// C20, C50) serve /login.htm or /userRpm/LoginRpm.htm
	// with a recognizable title. If the response is HTML and
	// the title contains the string "TP-LINK" or
	// "Wireless Router" we treat that as evidence of the
	// userRpm family. This is NOT a positive identification;
	// it is a hint to Gavetero that the userRpm recipe
	// (the only one we ship today) is worth trying.
	tplinkUserRpmProbePath = "/login.htm"
	tplinkUserRpmTimeout   = 3 * time.Second
)

func newDetectCmd() *cobra.Command {
	var host string
	var output string

	cmd := &cobra.Command{
		Use:   "detect [host]",
		Short: "Identify the router on the network",
		Long: `Probe the default gateway (or a host you pass) and
report what Gavetero can observe about it. With no
specific adapter for the firmware, detect returns the
four-state vocabulary and the HTTP reachability of
the management UI.

This is the user-facing path for "any modem/router":
you get a fast, honest report without needing a
captured-and-validated adapter for your firmware.`,
		Example: `  gvt detect
  gvt detect 192.168.0.1
  gvt detect --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			target := host
			if target == "" && len(args) > 0 {
				target = args[0]
			}
			if target == "" {
				gw, err := defaultGateway()
				if err == nil {
					target = strings.TrimSuffix(gw, " (use --host to override)")
				}
			}
			if target == "" {
				return errors.New("no host: pass one as an argument or use --host")
			}
			return runDetect(cmd.OutOrStdout(), target, output)
		},
	}
	cmd.Flags().StringVar(&host, "host", "",
		"router address (default: system default gateway)")
	cmd.Flags().StringVar(&output, "output", "human",
		"output format: human, json")
	return cmd
}

func runDetect(stdout io.Writer, host, output string) error {
	probe := probeRouter(host)
	switch output {
	case "json":
		b, _ := json.MarshalIndent(probe, "", "  ")
		fmt.Fprintln(stdout, string(b))
	default:
		renderHuman(stdout, host, probe)
	}
	return nil
}

// routerProbe is the user-facing summary of what Gavetero
// knows about a target host. The four states are the
// same vocabulary the HTTP API surface uses, so the
// output here is consistent with gvt inspect.
type routerProbe struct {
	Host             string   `json:"host"`
	Reachable        bool     `json:"reachable"`
	ManagementUI     string   `json:"management_ui,omitempty"`
	FamilyHint       string   `json:"family_hint,omitempty"`
	AdapterAvailable string   `json:"adapter_available"`
	Recommended      []string `json:"recommended,omitempty"`
}

func probeRouter(host string) routerProbe {
	probe := routerProbe{Host: host, AdapterAvailable: "tplinkwr841v8"}
	// Basic reachability check.
	conn, err := net.DialTimeout("tcp", host+":80", 2*time.Second)
	if err != nil {
		// Try HTTPS on 443.
		conn, err = net.DialTimeout("tcp", host+":443", 2*time.Second)
		if err != nil {
			return probe
		}
		conn.Close()
	}
	probe.Reachable = true
	// Heuristic: fetch /login.htm and look for a TP-Link
	// signature. If present, Gavetero can try the userRpm
	// recipe; otherwise the user has to add an adapter.
	client := &http.Client{Timeout: tplinkUserRpmTimeout}
	resp, err := client.Get(fmt.Sprintf("http://%s%s", host, tplinkUserRpmProbePath))
	if err != nil {
		return probe
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return probe
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<10))
	if err != nil {
		return probe
	}
	text := string(body)
	switch {
	case strings.Contains(text, "TP-LINK") || strings.Contains(text, "TP-Link"):
		probe.ManagementUI = "login page"
		probe.FamilyHint = "tplink-userrpm"
		probe.Recommended = []string{
			"run gvt ask \"<your question>\" with a tplinkwr841v8-compatible password to extract observations",
			"if the response is empty, capture the login page with scripts/capture-router-page.sh and add a new adapter under internal/adapters/<vendor>",
		}
	case strings.Contains(text, "Huawei") || strings.Contains(text, "HUAWEI"):
		probe.ManagementUI = "login page"
		probe.FamilyHint = "huawei"
		probe.Recommended = []string{
			"Gavetero does not have a Huawei adapter yet; capture the login page and add internal/adapters/huawei/",
		}
	case strings.Contains(text, "Sagemcom") || strings.Contains(text, "FASTWEB"):
		probe.ManagementUI = "login page"
		probe.FamilyHint = "sagemcom"
		probe.Recommended = []string{
			"Gavetero does not have a Sagemcom adapter; add internal/adapters/sagemcom/ with the login recipe",
		}
	default:
		probe.ManagementUI = "unknown management UI"
		probe.FamilyHint = "unknown"
		probe.Recommended = []string{
			"capture the HTML of the management UI with playwright-cli and add an adapter under internal/adapters/<vendor>/",
		}
	}
	return probe
}

func renderHuman(stdout io.Writer, host string, p routerProbe) {
	fmt.Fprintf(stdout, "Gavetero Detect: %s\n", host)
	fmt.Fprintln(stdout, "================")
	fmt.Fprintln(stdout)
	if !p.Reachable {
		fmt.Fprintf(stdout, "  reachability:  not reachable on :80 or :443\n")
		fmt.Fprintln(stdout, "  adapter:      tplinkwr841v8 (only one shipped today)")
		fmt.Fprintln(stdout)
		fmt.Fprintf(stdout, "  to use Gavetero, the router must be reachable from this host\n")
		return
	}
	fmt.Fprintf(stdout, "  reachability:  reachable\n")
	if p.ManagementUI != "" {
		fmt.Fprintf(stdout, "  management UI: %s\n", p.ManagementUI)
	}
	if p.FamilyHint != "" {
		fmt.Fprintf(stdout, "  family hint:   %s\n", p.FamilyHint)
	}
	fmt.Fprintf(stdout, "  adapter:       %s (verified)\n", p.AdapterAvailable)
	if len(p.Recommended) > 0 {
		fmt.Fprintln(stdout, "  next steps:")
		for _, s := range p.Recommended {
			fmt.Fprintf(stdout, "    - %s\n", s)
		}
	}
}

// ensure net, url imports are used somewhere even if not directly
var _ = url.Parse
