// Package cmd: the integrations subcommand. Copies the
// portable Gavetero skill into the install directory of the
// agent the user wants to use it with. Currently supported:
//
//	hermes      -> ~/.hermes/skills/gavetero/
//	opencode     -> ~/.config/opencode/skills/gavetero/

// The install is a plain file copy. The user does not need
// to restart the agent to pick up the skill (most agents
// load SKILL.md lazily on first use, but Hermes in
// particular announces the skill on session start).
package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	skill "github.com/Quiarom/gavetero/skills/gavetero"
	"github.com/spf13/cobra"
)

func newIntegrationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "integrations <install|list> [target]",
		Short: "Install the Gavetero skill into an agent's skill directory",
		Long: `Manage Gavetero integrations with other agents.

Subcommands:

  gvt integrations install <hermes|opencode>
      Copy the embedded Gavetero skill into the agent's skill
      directory. The skill becomes available as a slash
      command (Hermes) or as a name-activatable tool
      (OpenCode, OMP).

  gvt integrations list
      List the install directories known to gavetero and
      whether the skill is currently present.`,
		Example: `  gvt integrations install opencode
  gvt integrations install hermes
  gvt integrations list`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "install":
				if len(args) < 2 {
					return errors.New("usage: gvt integrations install <hermes|opencode>")
				}
				return runIntegrationsInstall(cmd.OutOrStdout(), args[1])
			case "list":
				return runIntegrationsList(cmd.OutOrStdout())
			default:
				return fmt.Errorf("unknown subcommand %q (want install or list)", args[0])
			}
		},
	}
	return cmd
}

func runIntegrationsInstall(stdout io.Writer, target string) error {
	dest, err := integrationTargetDir(target)
	if err != nil {
		return err
	}
	body, err := skill.Files.ReadFile("SKILL.md")
	if err != nil {
		return fmt.Errorf("read embedded skill: %w", err)
	}
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}
	dstFile := filepath.Join(dest, "SKILL.md")
	if err := os.WriteFile(dstFile, body, 0644); err != nil {
		return fmt.Errorf("write %s: %w", dstFile, err)
	}
	fmt.Fprintf(stdout, "\u2713 Installed Gavetero skill to %s\n", dstFile)
	return nil
}

func runIntegrationsList(stdout io.Writer) error {
	targets := []struct {
		name string
		path string
	}{
		{"hermes", filepath.Join(homeDir(), ".hermes", "skills", "gavetero", "SKILL.md")},
		{"opencode", filepath.Join(homeDir(), ".config", "opencode", "skills", "gavetero", "SKILL.md")},
	}
	fmt.Fprintln(stdout, "Gavetero skill install status:")
	fmt.Fprintln(stdout, "------------------------------")
	for _, t := range targets {
		_, err := os.Stat(t.path)
		state := "not installed"
		if err == nil {
			state = "installed"
		}
		fmt.Fprintf(stdout, "  %-10s %-12s %s\n", t.name, state, t.path)
	}
	return nil
}

func integrationTargetDir(target string) (string, error) {
	home := homeDir()
	switch target {
	case "hermes":
		return filepath.Join(home, ".hermes", "skills", "gavetero"), nil
	case "opencode":
		return filepath.Join(home, ".config", "opencode", "skills", "gavetero"), nil
	default:
		return "", fmt.Errorf("unknown integration target %q (want hermes or opencode)", target)
	}
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}
