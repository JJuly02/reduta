package httpserver

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/JJuly02/reduta/internal/store"
)

func (s *Server) handleCreateEvent(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r)
	var in struct {
		Slug            string  `json:"slug"`
		Name            string  `json:"name"`
		StartsAt        *string `json:"starts_at"`
		EndsAt          *string `json:"ends_at"`
		FreezeAt        *string `json:"freeze_at"`
		FirstBloodBonus int     `json:"first_blood_bonus"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	in.Slug = strings.TrimSpace(in.Slug)
	if in.Slug == "" || strings.TrimSpace(in.Name) == "" {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "slug and name are required")
		return
	}
	startsAt, err1 := parseTimePtr(in.StartsAt)
	endsAt, err2 := parseTimePtr(in.EndsAt)
	freezeAt, err3 := parseTimePtr(in.FreezeAt)
	if err1 != nil || err2 != nil || err3 != nil {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "timestamps must be RFC3339")
		return
	}
	e, err := s.store.CreateEvent(r.Context(), u.OrgID, in.Slug, in.Name, startsAt, endsAt, freezeAt, in.FirstBloodBonus)
	if errors.Is(err, store.ErrConflict) {
		s.writeProblem(w, http.StatusConflict, "Conflict", "event slug already exists")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "create event failed")
		return
	}
	s.writeJSON(w, http.StatusCreated, e)
}

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r)
	orgID, err := s.store.DefaultOrgID(r.Context())
	if u != nil {
		orgID = u.OrgID
	} else if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "no default org")
		return
	}
	isAdmin := u != nil && (u.Role == "owner" || u.Role == "admin")
	var events []store.Event
	if u != nil && !isAdmin {
		// players see only events their team is assigned to
		t, terr := s.store.UserTeam(r.Context(), u.ID)
		if terr != nil {
			events = []store.Event{}
		} else {
			events, err = s.store.ListEventsForTeam(r.Context(), orgID, t.ID)
		}
	} else {
		events, err = s.store.ListEvents(r.Context(), orgID)
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "list failed")
		return
	}
	if events == nil {
		events = []store.Event{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (s *Server) handleGetEvent(w http.ResponseWriter, r *http.Request) {
	e, err := s.store.GetEvent(r.Context(), chi.URLParam(r, "eventID"))
	if errors.Is(err, store.ErrNotFound) {
		s.writeProblem(w, http.StatusNotFound, "Not found", "event not found")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "get failed")
		return
	}
	s.writeJSON(w, http.StatusOK, e)
}

func (s *Server) handleSetEventState(w http.ResponseWriter, r *http.Request) {
	var in struct {
		State string `json:"state"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	if !oneOf(in.State, "draft", "running", "ended") {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "state must be draft|running|ended")
		return
	}
	err := s.store.SetEventState(r.Context(), chi.URLParam(r, "eventID"), in.State)
	if errors.Is(err, store.ErrNotFound) {
		s.writeProblem(w, http.StatusNotFound, "Not found", "event not found")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "update failed")
		return
	}
	s.publish(chi.URLParam(r, "eventID"), "challenges.changed", nil)
	s.writeJSON(w, http.StatusOK, map[string]string{"state": in.State})
}
