// Package schedule decides whether a challenge/block is open at a given time,
// expanding RRULE windows on the fly (spec 5.2). Empty schedule = always open.
package schedule

import (
	"encoding/json"
	"regexp"
	"strconv"
	"time"

	"github.com/teambition/rrule-go"
)

type Window struct {
	OpensAt  string `json:"opens_at"` // RFC3339
	Duration string `json:"duration"` // ISO-8601 (e.g. PT4H); empty = open-ended
	RRule    string `json:"rrule"`    // RFC 5545 (e.g. FREQ=DAILY;COUNT=5); empty = one-off
}

type Schedule struct {
	Timezone       string   `json:"timezone"`
	Windows        []Window `json:"windows"`
	ClosedBehavior string   `json:"closed_behavior"` // hidden|locked|readonly (default hidden)
}

// OpenAt reports whether the schedule is open at t and, when closed, the
// configured closed behavior. An empty/invalid schedule is always open.
func OpenAt(raw []byte, t time.Time) (open bool, closedBehavior string) {
	if len(raw) == 0 || string(raw) == "null" {
		return true, ""
	}
	var s Schedule
	if json.Unmarshal(raw, &s) != nil || len(s.Windows) == 0 {
		return true, ""
	}
	cb := s.ClosedBehavior
	if cb == "" {
		cb = "hidden"
	}
	for _, w := range s.Windows {
		start, err := time.Parse(time.RFC3339, w.OpensAt)
		if err != nil {
			continue
		}
		dur := parseISODuration(w.Duration)
		if w.RRule == "" {
			if within(t, start, dur) {
				return true, cb
			}
			continue
		}
		if dur <= 0 {
			dur = 24 * time.Hour
		}
		opt, err := rrule.StrToROption(w.RRule)
		if err != nil {
			continue
		}
		opt.Dtstart = start
		rr, err := rrule.NewRRule(*opt)
		if err != nil {
			continue
		}
		for _, occ := range rr.Between(t.Add(-dur), t, true) {
			if within(t, occ, dur) {
				return true, cb
			}
		}
	}
	return false, cb
}

func within(t, start time.Time, dur time.Duration) bool {
	if t.Before(start) {
		return false
	}
	if dur <= 0 {
		return true // open-ended from start
	}
	return t.Before(start.Add(dur))
}

var isoRe = regexp.MustCompile(`^P(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?)?$`)

func parseISODuration(s string) time.Duration {
	if s == "" {
		return 0
	}
	m := isoRe.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	at := func(x string) int { n, _ := strconv.Atoi(x); return n }
	return time.Duration(at(m[1]))*24*time.Hour +
		time.Duration(at(m[2]))*time.Hour +
		time.Duration(at(m[3]))*time.Minute +
		time.Duration(at(m[4]))*time.Second
}
