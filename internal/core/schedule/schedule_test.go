package schedule

import (
	"testing"
	"time"
)

func TestOpenAt(t *testing.T) {
	if open, _ := OpenAt(nil, time.Now()); !open {
		t.Error("empty schedule must be open")
	}
	// one-off window 2026-09-12 18:00Z for 4h
	sched := []byte(`{"windows":[{"opens_at":"2026-09-12T18:00:00Z","duration":"PT4H"}],"closed_behavior":"locked"}`)
	inside := time.Date(2026, 9, 12, 19, 0, 0, 0, time.UTC)
	before := time.Date(2026, 9, 12, 17, 0, 0, 0, time.UTC)
	after := time.Date(2026, 9, 12, 23, 0, 0, 0, time.UTC)
	if open, _ := OpenAt(sched, inside); !open {
		t.Error("should be open inside window")
	}
	if open, cb := OpenAt(sched, before); open || cb != "locked" {
		t.Errorf("should be closed(locked) before, got open=%v cb=%q", open, cb)
	}
	if open, _ := OpenAt(sched, after); open {
		t.Error("should be closed after window")
	}
	// daily recurrence for 3 days, 2h each
	rr := []byte(`{"windows":[{"opens_at":"2026-09-12T18:00:00Z","duration":"PT2H","rrule":"FREQ=DAILY;COUNT=3"}]}`)
	day2 := time.Date(2026, 9, 13, 18, 30, 0, 0, time.UTC)
	day2closed := time.Date(2026, 9, 13, 21, 0, 0, 0, time.UTC)
	if open, _ := OpenAt(rr, day2); !open {
		t.Error("should be open on day2 within recurrence")
	}
	if open, _ := OpenAt(rr, day2closed); open {
		t.Error("should be closed on day2 outside window")
	}
}

func TestParseISODuration(t *testing.T) {
	cases := map[string]time.Duration{
		"PT4H":     4 * time.Hour,
		"PT30M":    30 * time.Minute,
		"P1DT2H":   26 * time.Hour,
		"":         0,
		"garbage":  0,
	}
	for in, want := range cases {
		if got := parseISODuration(in); got != want {
			t.Errorf("parseISODuration(%q)=%v want %v", in, got, want)
		}
	}
}
