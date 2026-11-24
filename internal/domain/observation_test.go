package domain

import (
	"encoding/json"
	"testing"
)

func TestDecodeObservation(t *testing.T) {
	for _, tc := range []struct {
		body string
		ok   bool
	}{
		{`{"state":"mock","reason":"test data","mode":"mock","provenance":"fixture","data":{"value":true}}`, true},
		{`{"state":"absent","reason":"empty test data","mode":"mock","provenance":"fixture","data":[]}`, true},
		{`{"state":"unavailable","reason":"read failed","mode":"mock","provenance":"fixture","data":null}`, true},
		{`{"state":"verified","reason":"read","provenance":"observed","data":{"value":true}}`, true},
		{`{"state":"mock","reason":"test data","provenance":"fixture","data":{"value":true}}`, false},
		{`{"state":"mock","reason":"test data","mode":"mock","provenance":"observed","data":{"value":true}}`, false},
		{`{"state":"verified","reason":"test data","mode":"mock","provenance":"fixture","data":{"value":true}}`, false},
		{`{"state":"mock","reason":"test data","mode":"mock","provenance":"fixture","data":null}`, false},
	} {
		t.Run(tc.body, func(t *testing.T) {
			got, err := DecodeObservation([]byte(tc.body))
			if (err == nil) != tc.ok {
				t.Fatalf("got=%+v err=%v", got, err)
			}
		})
	}
	t.Run("live encoding unchanged", func(t *testing.T) {
		want := `{"state":"verified","reason":"read","provenance":"observed","data":{"value":true}}`
		value, err := DecodeObservation([]byte(want))
		if err != nil {
			t.Fatal(err)
		}
		got, err := json.Marshal(value)
		if err != nil || string(got) != want {
			t.Fatalf("live encoding=%s err=%v", got, err)
		}
	})
}
