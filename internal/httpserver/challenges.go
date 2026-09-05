package httpserver

import (
	"encoding/json"
	"errors"
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

type challengeListItem struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Category string   `json:"category"`
	State    string   `json:"state"`
	Points   int      `json:"points"`
	Position int      `json:"position"`
	BlockID  *string  `json:"block_id"`
	Tags     []string `json:"tags"`
	Locked   bool     `json:"locked"`
	Status   string   `json:"status"` // open|locked|scheduled
	Solves   int      `json:"solves"`
}

type challengeDetail struct {
	ID            string `json:"id"`
	EventID       string `json:"event_id"`
	Title         string `json:"title"`
	Category      string `json:"category"`
	DescriptionMD string `json:"description_md"`
	State         string `json:"state"`
	Points        int    `json:"points"`
}

func (s *Server) handleCreateChallenge(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	if _, err := s.store.GetEvent(r.Context(), eventID); errors.Is(err, store.ErrNotFound) {
		s.writeProblem(w, http.StatusNotFound, "Not found", "event not found")
		return
	}
	var in struct {
		Title         string          `json:"title"`
		Category      string          `json:"category"`
		DescriptionMD string          `json:"description_md"`
		Scoring       json.RawMessage `json:"scoring"`
		State         string          `json:"state"`
		Flags         []struct {
			Value         string `json:"value"`
			CaseSensitive *bool  `json:"case_sensitive"`
		} `json:"flags"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	if strings.TrimSpace(in.Title) == "" {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "title is required")
		return
	}
	if len(in.Flags) == 0 {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "at least one flag is required")
		return
	}
	if in.Category == "" {
		in.Category = "misc"
	}
	if in.State == "" {
		in.State = "draft"
	}
	if !oneOf(in.State, "draft", "published", "hidden") {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "state must be draft|published|hidden")
		return
	}
	if len(in.Scoring) == 0 {
		in.Scoring = json.RawMessage(`{"type":"static","points":100}`)
	}
	c, err := s.store.CreateChallenge(r.Context(), eventID, strings.TrimSpace(in.Title), in.Category, in.DescriptionMD, in.Scoring, in.State)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "create challenge failed")
		return
	}
	for _, f := range in.Flags {
		cs := true
		if f.CaseSensitive != nil {
			cs = *f.CaseSensitive
		}
		if strings.TrimSpace(f.Value) == "" {
			continue
		}
		if err := s.store.AddFlag(r.Context(), c.ID, flagpkg.Hash(f.Value, cs), cs); err != nil {
			s.writeProblem(w, http.StatusInternalServerError, "Internal error", "add flag failed")
			return
		}
	}
	s.writeJSON(w, http.StatusCreated, challengeDetail{
		ID: c.ID, EventID: c.EventID, Title: c.Title, Category: c.Category,
		DescriptionMD: c.DescriptionMD, State: c.State, Points: scoring.Points(c.Scoring),
	})
}

func (s *Server) handleListChallenges(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	u, _ := userFrom(r)
	isAdmin := u != nil && (u.Role == "owner" || u.Role == "admin")
	list, err := s.store.ListChallenges(r.Context(), eventID, !isAdmin)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "list failed")
		return
	}
	// Block map for schedule/unlock inheritance.
	blocks, _ := s.store.ListBlocks(r.Context(), eventID)
	bmap := map[string]store.Block{}
	for _, b := range blocks {
		bmap[b.ID] = b
	}

	// Per-team state for unlock evaluation (players on a team).
	ts := unlock.TeamState{Now: time.Now(), SolvedEC: map[string]bool{}, BlockSolved: map[string]int{}, BlockTotal: map[string]int{}}
	if !isAdmin && u != nil {
		if teamID, err := s.store.PlayerTeamForEvent(r.Context(), eventID, u.ID); err == nil {
			if gs, err := s.store.TeamGateState(r.Context(), eventID, teamID); err == nil {
				for _, e := range gs.SolvedEC {
					ts.SolvedEC[e] = true
				}
				ts.Points, ts.SolvedTotal, ts.BlockSolved, ts.BlockTotal = gs.Points, len(gs.SolvedEC), gs.BlockSolved, gs.BlockTotal
			}
		}
	}

	counts, _ := s.store.SolveCounts(r.Context(), eventID)

	items := make([]challengeListItem, 0, len(list))
	for _, c := range list {
		if c.Tags == nil {
			c.Tags = []string{}
		}
		locked, status := false, "open"
		if !isAdmin {
			sched := c.Schedule
			var blockUnlock []byte
			if c.BlockID != nil {
				if b, ok := bmap[*c.BlockID]; ok {
					if len(sched) == 0 {
						sched = b.Schedule
					}
					blockUnlock = b.UnlockRule
				}
			}
			if open, cb := schedule.OpenAt(sched, ts.Now); !open {
				if cb == "hidden" {
					continue
				}
				locked, status = true, "scheduled"
			}
			if !unlock.Unlocked(c.UnlockRule, ts) || !unlock.Unlocked(blockUnlock, ts) {
				locked, status = true, "locked"
			}
		}
		items = append(items, challengeListItem{
			ID: c.ID, Title: c.Title, Category: c.Category, State: c.State,
			Points: scoring.Points(c.Scoring), Position: c.Position,
			BlockID: c.BlockID, Tags: c.Tags, Locked: locked, Status: status,
			Solves: counts[c.ID],
		})
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"challenges": items})
}

func (s *Server) handleGetChallenge(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r)
	isAdmin := u != nil && (u.Role == "owner" || u.Role == "admin")
	c, err := s.store.GetChallenge(r.Context(), chi.URLParam(r, "ecID"))
	if errors.Is(err, store.ErrNotFound) {
		s.writeProblem(w, http.StatusNotFound, "Not found", "challenge not found")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "get failed")
		return
	}
	if c.State != "published" && !isAdmin {
		s.writeProblem(w, http.StatusNotFound, "Not found", "challenge not found")
		return
	}
	s.writeJSON(w, http.StatusOK, challengeDetail{
		ID: c.ID, EventID: c.EventID, Title: c.Title, Category: c.Category,
		DescriptionMD: c.DescriptionMD, State: c.State, Points: scoring.Points(c.Scoring),
	})
}

func (s *Server) handleSetChallengeState(w http.ResponseWriter, r *http.Request) {
	var in struct {
		State string `json:"state"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	if !oneOf(in.State, "draft", "published", "hidden") {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "state must be draft|published|hidden")
		return
	}
	err := s.store.SetChallengeState(r.Context(), chi.URLParam(r, "ecID"), in.State)
	if errors.Is(err, store.ErrNotFound) {
		s.writeProblem(w, http.StatusNotFound, "Not found", "challenge not found")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "update failed")
		return
	}
	s.publish(chi.URLParam(r, "eventID"), "challenges.changed", nil)
	s.writeJSON(w, http.StatusOK, map[string]string{"state": in.State})
}

// handleMyAttempts returns the calling player's team submissions for a challenge.
// Raw wrong flags are never stored, so each row carries only a verdict + time.
func (s *Server) handleMyAttempts(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r)
	eventID := chi.URLParam(r, "eventID")
	ecID := chi.URLParam(r, "ecID")
	teamID, err := s.store.PlayerTeamForEvent(r.Context(), eventID, u.ID)
	if errors.Is(err, store.ErrNotFound) {
		s.writeJSON(w, http.StatusOK, map[string]any{"attempts": []store.AttemptRow{}})
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "team lookup failed")
		return
	}
	att, err := s.store.TeamChallengeAttempts(r.Context(), ecID, teamID)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "attempts failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"attempts": att})
}
