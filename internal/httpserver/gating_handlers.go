package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/JJuly02/reduta/internal/store"
)

func (s *Server) handleSetChallengeSchedule(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Schedule json.RawMessage `json:"schedule"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	err := s.store.SetChallengeSchedule(r.Context(), chi.URLParam(r, "ecID"), in.Schedule)
	if err == nil {
		s.publish(chi.URLParam(r, "eventID"), "challenges.changed", nil)
	}
	s.respondUpdate(w, err)
}

func (s *Server) handleSetChallengeUnlock(w http.ResponseWriter, r *http.Request) {
	var in struct {
		UnlockRule json.RawMessage `json:"unlock_rule"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	err := s.store.SetChallengeUnlock(r.Context(), chi.URLParam(r, "ecID"), in.UnlockRule)
	if err == nil {
		s.publish(chi.URLParam(r, "eventID"), "challenges.changed", nil)
	}
	s.respondUpdate(w, err)
}

func (s *Server) handleSetBlockSchedule(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Schedule json.RawMessage `json:"schedule"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	err := s.store.SetBlockSchedule(r.Context(), chi.URLParam(r, "blockID"), in.Schedule)
	s.respondUpdate(w, err)
}

func (s *Server) respondUpdate(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		s.writeProblem(w, http.StatusNotFound, "Not found", "not found")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "update failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
