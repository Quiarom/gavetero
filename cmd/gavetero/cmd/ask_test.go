package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

func TestAskDryRunUsesCLIKeyEnvironment(t *testing.T) {
	t.Setenv("GMI_KEY_ENV", "OPENROUTER_API_KEY")
	t.Setenv("OPENROUTER_API_KEY", "ignored-synthetic-key")
	t.Setenv("GMI_SERVING_API_KEY", "synthetic-gmi-key")

	var out bytes.Buffer
	err := runAsk(&out, io.Discard, askOptions{
		Question: "Wi-Fi?",
		Output:   "jsonl",
		MockMode: true,
		DryRun:   true,
		Model:    defaultModel,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Answer string
		Mode   string
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Mode != "mock" || !strings.Contains(result.Answer, "GMI_SERVING_API_KEY") ||
		strings.Contains(result.Answer, "OPENROUTER_API_KEY") {
		t.Fatalf("unexpected dry-run response: mode=%q answer=%q", result.Mode, result.Answer)
	}
	keyEnv, key, err := resolveAPIKey()
	if err != nil || keyEnv != "GMI_SERVING_API_KEY" || key != "synthetic-gmi-key" {
		t.Fatal("ask did not resolve the credential named in its dry-run response")
	}
}
