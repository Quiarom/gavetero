package cmd

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Quiarom/gavetero/internal/adapters/fixture"
)

func TestPrintAdapterJSON(t *testing.T) {
	for _, value := range []any{map[string]any{"present": true}, map[string]any{}, map[string]any{"provenance": "absent"}} {
		func() {
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			previous := os.Stdout
			os.Stdout = writer
			defer func() { os.Stdout = previous }()
			err = printAdapterJSON(fixture.New(""), value)
			writer.Close()
			body, readErr := io.ReadAll(reader)
			var got map[string]any
			if err != nil || readErr != nil || json.Unmarshal(body, &got) != nil || got["mode"] != "mock" || got["provenance"] != "fixture" || strings.Contains(string(body), `"verified"`) {
				t.Fatalf("mock JSON=%s errors=%v,%v", body, err, readErr)
			}
		}()
	}
	if err := printAdapterJSON(fixture.New(""), make(chan int)); err == nil {
		t.Fatal("expected encoding error")
	}
}
