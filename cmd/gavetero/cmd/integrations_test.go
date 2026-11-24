package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	skill "github.com/Quiarom/gavetero/skills/gavetero"
)

func TestRunIntegrationsInstall(t *testing.T) {
	t.Chdir(t.TempDir())
	want, err := skill.Files.ReadFile("SKILL.md")
	if err != nil || len(want) == 0 {
		t.Fatalf("embedded skill: %v", err)
	}
	for _, target := range []string{"hermes", "opencode"} {
		t.Run(target, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			var out bytes.Buffer
			if err := runIntegrationsInstall(&out, target); err != nil {
				t.Fatal(err)
			}
			dir, err := integrationTargetDir(target)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "SKILL.md")
			got, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(got, want) || !strings.Contains(out.String(), path) {
				t.Fatalf("installed skill mismatch: %v; output=%s", err, out.String())
			}
		})
	}
	t.Run("unknown target", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		if err := runIntegrationsInstall(&bytes.Buffer{}, ""); err == nil {
			t.Fatal("expected an unknown target error")
		}
	})
	t.Run("unwritable destination", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		if err := os.WriteFile(filepath.Join(home, ".hermes"), []byte("not a directory"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := runIntegrationsInstall(&bytes.Buffer{}, "hermes"); err == nil {
			t.Fatal("expected a destination error")
		}
	})
}
