package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunDoctor(t *testing.T) {
	var out, stderr bytes.Buffer
	if err := runDoctor(&out, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Installation checks only; no router observations.",
		"mock test data, not a real router",
		"not a check of your router",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %q in %s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "verified adapter") {
		t.Fatal("doctor claimed adapter verification")
	}
}
