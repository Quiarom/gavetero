package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestExecuteQuestion_Mock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"state":"mock","mode":"mock","reason":"test data","provenance":"fixture","data":{"present":true}}`)
	}))
	defer srv.Close()
	got, err := executeQuestion(t.Context(), options{routerCoreURL: srv.URL, timeout: time.Second, dryRun: true}, "¿Quién está conectado?")
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(got)
	if err != nil || got.Mode != "mock" || got.Provenance != "fixture" || len(got.Steps) != 6 || strings.Contains(string(body), `"verified"`) {
		t.Fatalf("mock result=%s err=%v", body, err)
	}
}
