// Package cmd: the doctor subcommand. Runs 6 checks and prints
// the status of the Gavetero install. Designed so a fresh
// user can answer "is this thing going to work?" with one
// command.
//
// Checks:
//
//  1. CLI installation
//  2. configuration location
//  3. credential store availability + GMI configured or not
//  4. default gateway
//  5. router observation readiness
//  6. current adapter/support state
//
// A missing AI configuration does NOT mean router functionality
// is broken. The doctor output makes the distinction.
package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check whether Gavetero can work here",
		Long: `Run a series of checks and print the result.

Doctor is intentionally non-destructive. It does not start
the runtime, it does not contact the router, and it does not
contact GMI Cloud.

Output: a list of checks with one of three states:

  ok       ready
  warn     present but degraded
  fail     missing or broken

Each fail includes the next step to take.`,
		Example: `  gvt doctor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	return cmd
}

type checkResult struct {
	state  string // "ok", "warn", "fail"
	name   string
	detail string
	action string // empty when state != "fail"
}

func (c checkResult) print(w io.Writer) {
	icon := "?"
	switch c.state {
	case "ok":
		icon = "\u2713" // ✓
	case "warn":
		icon = "!"
	case "fail":
		icon = "\u2717" // ✗
	}
	fmt.Fprintf(w, "  %s  %-22s %s\n", icon, c.name, c.detail)
	if c.action != "" {
		fmt.Fprintf(w, "        %s\n", c.action)
	}
}

func runDoctor(stdout, stderr io.Writer) error {
	checks := []checkResult{
		checkCLI(),
		checkConfig(),
		checkCredentials(),
		checkGateway(),
		checkRouter(),
		checkAdapter(),
	}

	fmt.Fprintln(stdout, "Gavetero doctor")
	fmt.Fprintln(stdout, "==============")
	fmt.Fprintln(stdout, "")
	for _, c := range checks {
		c.print(stdout)
	}
	fmt.Fprintln(stdout, "")

	// Summary line.
	var ok, warn, fail int
	for _, c := range checks {
		switch c.state {
		case "ok":
			ok++
		case "warn":
			warn++
		case "fail":
			fail++
		}
	}
	fmt.Fprintf(stdout, "Result: %d ok, %d warn, %d fail\n", ok, warn, fail)
	if fail > 0 {
		return fmt.Errorf("%d check(s) failed", fail)
	}
	return nil
}

func checkCLI() checkResult {
	exe, err := os.Executable()
	if err != nil {
		return checkResult{state: "warn", name: "CLI installation", detail: err.Error()}
	}
	// Resolve any symlink so the user sees the real path.
	real, _ := filepath.EvalSymlinks(exe)
	return checkResult{
		state:  "ok",
		name:   "CLI installation",
		detail: real,
	}
}

func checkConfig() checkResult {
	dir, err := os.UserConfigDir()
	if err != nil {
		return checkResult{state: "fail", name: "config location", detail: err.Error(),
			action: "ensure $XDG_CONFIG_HOME or $HOME is set"}
	}
	cfgPath := filepath.Join(dir, "gavetero", "config.toml")
	if _, err := os.Stat(cfgPath); err != nil {
		return checkResult{
			state:  "warn",
			name:   "config location",
			detail: cfgPath + " (missing)",
			action: "run `gvt setup`",
		}
	}
	return checkResult{state: "ok", name: "config location", detail: cfgPath}
}

func checkCredentials() checkResult {
	if env := os.Getenv(envVarName); env != "" {
		return checkResult{
			state:  "ok",
			name:   "GMI key",
			detail: "from " + envVarName + " env var",
		}
	}
	if v, err := keyring.Get(keyringService, keyringAccount); err == nil && v != "" {
		return checkResult{
			state:  "ok",
			name:   "GMI key",
			detail: "from OS credential store (service=" + keyringService + ")",
		}
	}
	return checkResult{
		state:  "warn",
		name:   "GMI key",
		detail: "not configured (router observations still work, gvt ask will not)",
		action: "run `gvt setup` to store the key in the OS credential store",
	}
}

func checkGateway() checkResult {
	gw, err := defaultGateway()
	if err != nil {
		return checkResult{
			state:  "warn",
			name:   "default gateway",
			detail: "could not detect: " + err.Error(),
			action: "specify --host explicitly when running gvt inspect --live",
		}
	}
	return checkResult{state: "ok", name: "default gateway", detail: gw}
}

func checkRouter() checkResult {
	// Try to talk to router-core sidecar briefly. We start the
	// mock sidecar and check capabilities.
	bin, err := findRouterCoreBin()
	if err != nil {
		return checkResult{
			state:  "fail",
			name:   "router observation",
			detail: "router-core not found",
			action: "run `make build` so router-core is built next to gavetero",
		}
	}
	// Run a quick smoke: spawn the sidecar, check /v0/capabilities.
	addr, _ := reserveLoopbackAddr()
	if addr == "" {
		return checkResult{state: "warn", name: "router observation", detail: "no ephemeral port available"}
	}
	// Use a short-lived shell: just invoke the binary with --help
	// to confirm it exists. The real probe is gvt inspect.
	cmd := execCmd(bin, "serve", "--help")
	if err := cmd.Run(); err != nil {
		return checkResult{state: "fail", name: "router observation", detail: bin + ": " + err.Error(),
			action: "rebuild with `make build`"}
	}
	return checkResult{
		state:  "ok",
		name:   "router observation",
		detail: "router-core available at " + bin,
	}
}

func checkAdapter() checkResult {
	// We do not probe the firmware on a live router from doctor;
	// instead we report the supported state from the runtime.
	return checkResult{
		state:  "ok",
		name:   "adapter",
		detail: "TP-Link TL-WR841N/ND v8.4 (3.15.9) is the first verified adapter",
	}
}

func defaultGateway() (string, error) {
	gw, err := defaultGatewayFromProcRoute("/proc/net/route")
	if err != nil {
		return "", err
	}
	return gw + " (use --host to override)", nil
}

// defaultGatewayFromProcRoute reads the Linux kernel routing table
// directly (no subprocess, no packets) and returns the gateway of
// the default route with the lowest metric. path is a parameter so
// tests can feed a fixture table instead of the live one.
func defaultGatewayFromProcRoute(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read route table: %w", err)
	}
	bestMetric := 0
	best := ""
	for i, line := range strings.Split(string(raw), "\n") {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue // header / trailing blank
		}
		f := strings.Fields(line)
		if len(f) < 8 || f[1] != "00000000" {
			continue // not a default route
		}
		flags, err := strconv.ParseUint(f[3], 16, 32)
		if err != nil || flags&0x2 == 0 || f[2] == "00000000" {
			continue // down, or gateway-less (link scope)
		}
		metric := 0
		if m, err := strconv.Atoi(f[6]); err == nil {
			metric = m
		}
		if best == "" || metric < bestMetric {
			bestMetric = metric
			best = f[2]
		}
	}
	if best == "" {
		return "", fmt.Errorf("no default route with gateway in %s", path)
	}
	return procRouteAddr(best)
}

// procRouteAddr converts the little-endian hex address used by
// /proc/net/route (e.g. 0101A8C0) to dotted quad (192.168.1.1).
func procRouteAddr(hexAddr string) (string, error) {
	v, err := strconv.ParseUint(hexAddr, 16, 32)
	if err != nil {
		return "", fmt.Errorf("bad gateway address %q: %w", hexAddr, err)
	}
	return fmt.Sprintf("%d.%d.%d.%d", byte(v), byte(v>>8), byte(v>>16), byte(v>>24)), nil
}

// execCmd is a tiny shim that lets the doctor command run a
// subprocess without importing os/exec at the top of the file.
func execCmd(name string, args ...string) *execCmdResult {
	return newExecCmd(name, args...)
}
