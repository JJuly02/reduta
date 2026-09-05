// Package flags normalizes, hashes and verifies CTF flags. Raw flags are never
// stored; only sha256 of the normalized value is kept.
package flags

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
)

// Normalize trims and, when not case sensitive, lowercases the value.
func Normalize(v string, caseSensitive bool) string {
	v = strings.TrimSpace(v)
	if !caseSensitive {
		v = strings.ToLower(v)
	}
	return v
}

// Hash returns sha256 of the normalized value.
func Hash(v string, caseSensitive bool) []byte {
	sum := sha256.Sum256([]byte(Normalize(v, caseSensitive)))
	return sum[:]
}

// Spec is a stored flag: its hash and whether it was hashed case-sensitively.
type Spec struct {
	Hash          []byte
	CaseSensitive bool
}

// Verify reports whether submission matches any of the flags.
func Verify(submission string, specs []Spec) bool {
	cs := Hash(submission, true)
	ci := Hash(submission, false)
	ok := false
	for _, s := range specs {
		want := cs
		if !s.CaseSensitive {
			want = ci
		}
		if subtle.ConstantTimeCompare(s.Hash, want) == 1 {
			ok = true
		}
	}
	return ok
}

// TeamFlag derives a deterministic per-team flag (spec 5.8, {{team_flag}}), so a
// flag leaked by one team does not validate for another. HMAC over event/team/ec.
func TeamFlag(secret, eventID, teamID, ecID string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(eventID + ":" + teamID + ":" + ecID))
	return "flag{" + hex.EncodeToString(mac.Sum(nil))[:24] + "}"
}
