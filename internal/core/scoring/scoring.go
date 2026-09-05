// Package scoring computes challenge point values. M1 supports static scoring;
// dynamic decay is deferred to M5 (falls back to the initial value for now).
package scoring

import "encoding/json"

type Spec struct {
	Type    string `json:"type"`
	Points  int    `json:"points"`
	Initial int    `json:"initial"`
	Minimum int    `json:"minimum"`
	Decay   int    `json:"decay"`
}

// Points returns the value awarded for a solve given the scoring JSON.
func Points(raw []byte) int {
	var s Spec
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &s)
	}
	switch s.Type {
	case "static":
		return s.Points
	case "dynamic": // TODO(M5): real decay over solve count
		if s.Initial > 0 {
			return s.Initial
		}
		return s.Points
	case "manual":
		return 0
	default:
		return s.Points
	}
}
