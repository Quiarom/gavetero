package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type ObservationState string

const (
	Verified                ObservationState = "verified"
	Mock                    ObservationState = "mock"
	Absent                  ObservationState = "absent"
	UnsupportedOrUnverified ObservationState = "unsupported_or_unverified"
	Unavailable             ObservationState = "unavailable"
)

type Observation[T any] struct {
	State      ObservationState `json:"state"`
	Reason     string           `json:"reason"`
	Provenance Provenance       `json:"provenance"`
	Data       T                `json:"data"`
	Mode       string           `json:"mode,omitempty"`
}

func DecodeObservation(body []byte) (Observation[json.RawMessage], error) {
	var result Observation[json.RawMessage]
	if err := json.Unmarshal(body, &result); err != nil {
		return result, fmt.Errorf("decode observation: %w", err)
	}
	if strings.TrimSpace(result.Reason) == "" || len(result.Data) == 0 {
		return result, fmt.Errorf("observation requires a reason and data field")
	}
	switch result.Provenance {
	case ProvenanceObserved, ProvenanceFixture, ProvenanceAbsent:
	default:
		return result, fmt.Errorf("invalid observation provenance %q", result.Provenance)
	}
	switch result.State {
	case Verified, Mock:
		var compact bytes.Buffer
		if err := json.Compact(&compact, result.Data); err != nil {
			return result, fmt.Errorf("decode observation data: %w", err)
		}
		switch compact.String() {
		case "null", "{}", "[]", `""`:
			return result, fmt.Errorf("%s observation has no data", result.State)
		}
	case Absent, UnsupportedOrUnverified, Unavailable:
	default:
		return result, fmt.Errorf("invalid observation state %q", result.State)
	}
	if result.Mode == "mock" || result.State == Mock {
		if result.Mode != "mock" || result.Provenance != ProvenanceFixture || result.State == Verified {
			return result, fmt.Errorf("mock observation requires mode mock, fixture provenance, and a non-verified state")
		}
	}
	return result, nil
}
