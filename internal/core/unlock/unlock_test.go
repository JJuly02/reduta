package unlock

import (
	"testing"
	"time"
)

func TestEval(t *testing.T) {
	base := time.Date(2026, 9, 12, 20, 0, 0, 0, time.UTC)
	st := TeamState{
		SolvedEC:    map[string]bool{"a": true},
		Points:      600,
		SolvedTotal: 1,
		BlockSolved: map[string]int{"b1": 2},
		BlockTotal:  map[string]int{"b1": 2, "b2": 3},
		Now:         base,
	}
	p := func(n int) *int { return &n }
	s := func(x string) *string { return &x }
	tm := func(x time.Time) *time.Time { return &x }

	cases := []struct {
		name string
		rule Rule
		want bool
	}{
		{"empty unlocked", Rule{}, true},
		{"solved yes", Rule{Solved: &solvedRef{EC: "a"}}, true},
		{"solved no", Rule{Solved: &solvedRef{EC: "z"}}, false},
		{"points gte", Rule{TeamPointsGte: p(500)}, true},
		{"points gte fail", Rule{TeamPointsGte: p(700)}, false},
		{"after now", Rule{After: tm(base.Add(-time.Hour))}, true},
		{"after future", Rule{After: tm(base.Add(time.Hour))}, false},
		{"block completed", Rule{BlockCompleted: s("b1")}, true},
		{"block incomplete", Rule{BlockCompleted: s("b2")}, false},
		{"all pass", Rule{All: []Rule{{Solved: &solvedRef{EC: "a"}}, {TeamPointsGte: p(100)}}}, true},
		{"all fail", Rule{All: []Rule{{Solved: &solvedRef{EC: "a"}}, {TeamPointsGte: p(999)}}}, false},
		{"any pass", Rule{Any: []Rule{{TeamPointsGte: p(999)}, {Solved: &solvedRef{EC: "a"}}}}, true},
		{"not", Rule{Not: &Rule{Solved: &solvedRef{EC: "z"}}}, true},
	}
	for _, c := range cases {
		if got := Eval(c.rule, st); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestUnlockedRaw(t *testing.T) {
	st := TeamState{Points: 100}
	if !Unlocked(nil, st) {
		t.Error("nil rule should be unlocked")
	}
	if !Unlocked([]byte("null"), st) {
		t.Error("null rule should be unlocked")
	}
	if Unlocked([]byte(`{"team_points_gte":500}`), st) {
		t.Error("should be locked at 100 points")
	}
}
