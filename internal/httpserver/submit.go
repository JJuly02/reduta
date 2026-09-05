package httpserver

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	flagpkg "github.com/JJuly02/reduta/internal/core/flags"
	"github.com/JJuly02/reduta/internal/core/schedule"
	"github.com/JJuly02/reduta/internal/core/scoring"
	"github.com/JJuly02/reduta/internal/core/unlock"
	"github.com/JJuly02/reduta/internal/store"
)

func clientIP(r *http.Request) string {
	host := r.RemoteAddr
	if h, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		host = h
	}
	if net.ParseIP(host) == nil {
		return ""
	}
	return host
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r)
	eventID := chi.URLParam(r, "eventID")
	ecID := chi.URLParam(r, "ecID")

	e, err := s.store.GetEvent(r.Context(), eventID)
	if errors.Is(err, store.ErrNotFound) {
		s.writeProblem(w, http.StatusNotFound, "Not found", "event not found")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "event lookup failed")
		return
	}
	now := time.Now()
	if e.StartsAt != nil && now.Before(*e.StartsAt) {
		s.writeProblem(w, http.StatusForbidden, "Forbidden", "event has not started")
		return
	}
	if e.EndsAt != nil && now.After(*e.EndsAt) {
		s.writeProblem(w, http.StatusForbidden, "Forbidden", "event has ended")
		return
	}

	teamID, err := s.store.PlayerTeamForEvent(r.Context(), eventID, u.ID)
	if errors.Is(err, store.ErrNotFound) {
		s.writeProblem(w, http.StatusForbidden, "Forbidden", "your team is not registered for this event")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "membership lookup failed")
		return
	}

	if !s.rl.Allow("submit:" + eventID + ":" + teamID) {
		s.writeProblem(w, http.StatusTooManyRequests, "Too many requests", "submission rate limit exceeded")
		return
	}

	c, err := s.store.GetChallenge(r.Context(), ecID)
	if errors.Is(err, store.ErrNotFound) || (err == nil && (c.EventID != eventID || c.State != "published")) {
		s.writeProblem(w, http.StatusNotFound, "Not found", "challenge not found")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "challenge lookup failed")
		return
	}

	// M3: schedule + unlock gate (challenge schedule/unlock inherit from block).
	{
		sched := c.Schedule
		var blockUnlock []byte
		if c.BlockID != nil {
			if b, err := s.store.GetBlock(r.Context(), *c.BlockID); err == nil {
				if len(sched) == 0 {
					sched = b.Schedule
				}
				blockUnlock = b.UnlockRule
			}
		}
		if open, _ := schedule.OpenAt(sched, time.Now()); !open {
			s.writeProblem(w, http.StatusForbidden, "Forbidden", "challenge is not open")
			return
		}
		if gs, err := s.store.TeamGateState(r.Context(), eventID, teamID); err == nil {
			ts := unlock.TeamState{Now: time.Now(), SolvedEC: map[string]bool{}, BlockSolved: gs.BlockSolved, BlockTotal: gs.BlockTotal, Points: gs.Points, SolvedTotal: len(gs.SolvedEC)}
			for _, e := range gs.SolvedEC {
				ts.SolvedEC[e] = true
			}
			if !unlock.Unlocked(c.UnlockRule, ts) || !unlock.Unlocked(blockUnlock, ts) {
				s.writeProblem(w, http.StatusForbidden, "Forbidden", "challenge is locked")
				return
			}
		}
	}

	var in struct {
		Flag string `json:"flag"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	if in.Flag == "" {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "flag is required")
		return
	}

	dbFlags, err := s.store.ChallengeFlags(r.Context(), ecID)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "flag lookup failed")
		return
	}
	specs := make([]flagpkg.Spec, 0, len(dbFlags))
	for _, f := range dbFlags {
		specs = append(specs, flagpkg.Spec{Hash: f.ValueHash, CaseSensitive: f.CaseSensitive})
	}

	ip := clientIP(r)
	subHash := flagpkg.Hash(in.Flag, true)

	correct := flagpkg.Verify(in.Flag, specs)
	if !correct {
		// M8: per-team dynamic flag ({{team_flag}}) verification.
		if raw, err := s.store.ChallengeInstanceSpec(r.Context(), ecID); err == nil {
			if sp, has := parseInstanceSpec(raw); has && sp.usesTeamFlag() {
				if strings.TrimSpace(in.Flag) == flagpkg.TeamFlag(s.cfg.FlagSecret, eventID, teamID, ecID) {
					correct = true
				}
			}
		}
	}
	if !correct {
		if err := s.store.RecordWrong(r.Context(), eventID, ecID, teamID, u.ID, subHash, ip); err != nil {
			s.writeProblem(w, http.StatusInternalServerError, "Internal error", "record failed")
			return
		}
		s.writeJSON(w, http.StatusOK, map[string]any{"correct": false})
		return
	}

	res, err := s.store.RecordSolve(r.Context(), e, ecID, teamID, u.ID, subHash, ip, scoring.Points(c.Scoring))
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "record solve failed")
		return
	}
	if !res.AlreadySolved {
		// Realtime (M5): a new solve moves the scoreboard and may unlock challenges.
		s.scoreChanged(eventID)
		s.publish(eventID, "challenges.changed", nil)
		// Plugins (M7): notify subscribers of the solve.
		s.deliverWebhook("solve.created", map[string]any{"event_id": eventID, "team_id": teamID, "ec_id": ecID})
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"correct":        true,
		"already_solved": res.AlreadySolved,
		"first_blood":    res.FirstBlood,
		"points":         res.Points,
	})
}
