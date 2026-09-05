// Package unlock evaluates challenge unlock rules against a team's state.
// It is a pure function with no DB access (spec 5.3): the caller assembles
// TeamState. Predicates use event_challenge ids (ec) rather than library slugs
// — a documented simplification over spec's challenge_slug.
package unlock

import (
	"encoding/json"
	"time"
)

type TeamState struct {
	SolvedEC    map[string]bool // solved event_challenge ids
	Points      int
	SolvedTotal int
	BlockSolved map[string]int // block_id -> solved count for this team
	BlockTotal  map[string]int // block_id -> total published challenges
	Now         time.Time
}

// Rule is a JSON node; exactly one field is meaningful per node.
type Rule struct {
	All            []Rule     `json:"all,omitempty"`
	Any            []Rule     `json:"any,omitempty"`
	Not            *Rule      `json:"not,omitempty"`
	Solved         *solvedRef `json:"solved,omitempty"`
	TeamPointsGte  *int       `json:"team_points_gte,omitempty"`
	SolvedCountGte *int       `json:"solved_count_gte,omitempty"`
	After          *time.Time `json:"after,omitempty"`
	BlockCompleted *string    `json:"block_completed,omitempty"`
}

type solvedRef struct {
	EC string `json:"ec"`
}

func Eval(r Rule, st TeamState) bool {
	switch {
	case len(r.All) > 0:
		for _, c := range r.All {
			if !Eval(c, st) {
				return false
			}
		}
		return true
	case len(r.Any) > 0:
		for _, c := range r.Any {
			if Eval(c, st) {
				return true
			}
		}
		return false
	case r.Not != nil:
		return !Eval(*r.Not, st)
	case r.Solved != nil:
		return st.SolvedEC[r.Solved.EC]
	case r.TeamPointsGte != nil:
		return st.Points >= *r.TeamPointsGte
	case r.SolvedCountGte != nil:
		return st.SolvedTotal >= *r.SolvedCountGte
	case r.After != nil:
		return !st.Now.Before(*r.After)
	case r.BlockCompleted != nil:
		tot := st.BlockTotal[*r.BlockCompleted]
		return tot > 0 && st.BlockSolved[*r.BlockCompleted] >= tot
	default:
		return true // empty node = unlocked
	}
}

// Unlocked parses a raw unlock_rule (nil/empty/null = unlocked) and evaluates it.
func Unlocked(raw []byte, st TeamState) bool {
	if len(raw) == 0 || string(raw) == "null" {
		return true
	}
	var r Rule
	if err := json.Unmarshal(raw, &r); err != nil {
		return true
	}
	return Eval(r, st)
}
