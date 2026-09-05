package httpserver

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/JJuly02/reduta/internal/auth"
	"github.com/JJuly02/reduta/internal/store"
)

// handleCreateTeam creates the caller's global team; the creator is captain.
func (s *Server) handleCreateTeam(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r)
	var in struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "name is required")
		return
	}
	if _, err := s.store.UserTeam(r.Context(), u.ID); err == nil {
		s.writeProblem(w, http.StatusConflict, "Conflict", "you already belong to a team")
		return
	}
	code, err := auth.RandomToken(6)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "code gen failed")
		return
	}
	t, err := s.store.CreateTeam(r.Context(), u.OrgID, strings.TrimSpace(in.Name), code)
	if errors.Is(err, store.ErrConflict) {
		s.writeProblem(w, http.StatusConflict, "Conflict", "team name already taken")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "create team failed")
		return
	}
	if err := s.store.AddTeamMember(r.Context(), u.ID, t.ID, "captain"); err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "membership failed")
		return
	}
	t.Role = "captain"
	s.writeJSON(w, http.StatusCreated, t)
}

// handleJoinTeam joins the caller to a team by invite code.
func (s *Server) handleJoinTeam(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r)
	var in struct {
		InviteCode string `json:"invite_code"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	t, err := s.store.TeamByInvite(r.Context(), strings.TrimSpace(in.InviteCode))
	if errors.Is(err, store.ErrNotFound) {
		s.writeProblem(w, http.StatusNotFound, "Not found", "invalid invite code")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "lookup failed")
		return
	}
	err = s.store.AddTeamMember(r.Context(), u.ID, t.ID, "member")
	if errors.Is(err, store.ErrConflict) {
		s.writeProblem(w, http.StatusConflict, "Conflict", "you already belong to a team")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "join failed")
		return
	}
	t.InviteCode = ""
	t.Role = "member"
	s.writeJSON(w, http.StatusOK, t)
}

// handleMyTeam returns the caller's global team (invite code visible to members).
func (s *Server) handleMyTeam(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r)
	t, err := s.store.UserTeam(r.Context(), u.ID)
	if errors.Is(err, store.ErrNotFound) {
		s.writeJSON(w, http.StatusOK, map[string]any{"team": nil})
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "lookup failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"team": t})
}

// handleListTeams lists the teams assigned to an event.
func (s *Server) handleListTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := s.store.ListTeams(r.Context(), chi.URLParam(r, "eventID"))
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "list failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"teams": teams})
}

// handleMyStatus returns the caller's participation in an event.
func (s *Server) handleMyStatus(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r)
	eventID := chi.URLParam(r, "eventID")
	teamID, err := s.store.PlayerTeamForEvent(r.Context(), eventID, u.ID)
	if errors.Is(err, store.ErrNotFound) {
		s.writeJSON(w, http.StatusOK, map[string]any{"team_id": nil, "solved": []string{}, "points": 0})
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "status failed")
		return
	}
	gs, err := s.store.TeamGateState(r.Context(), eventID, teamID)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "status failed")
		return
	}
	solved := gs.SolvedEC
	if solved == nil {
		solved = []string{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"team_id": teamID, "solved": solved, "points": gs.Points})
}

// ---- admin: assign teams to events ----

func (s *Server) handleListOrgTeams(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r)
	teams, err := s.store.ListOrgTeams(r.Context(), u.OrgID)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "list failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"teams": teams})
}

func (s *Server) handleAssignEventTeam(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	var in struct {
		TeamID string `json:"team_id"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	if in.TeamID == "" {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "team_id is required")
		return
	}
	if err := s.store.AssignEventTeam(r.Context(), eventID, in.TeamID); err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "assign failed")
		return
	}
	s.scoreChanged(eventID)
	s.writeJSON(w, http.StatusOK, map[string]any{"assigned": true})
}

func (s *Server) handleUnassignEventTeam(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	if err := s.store.UnassignEventTeam(r.Context(), eventID, chi.URLParam(r, "teamID")); err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "unassign failed")
		return
	}
	s.scoreChanged(eventID)
	s.writeJSON(w, http.StatusOK, map[string]any{"removed": true})
}
