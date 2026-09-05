package httpserver

import (
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	flagpkg "github.com/JJuly02/reduta/internal/core/flags"
	"github.com/JJuly02/reduta/internal/store"
)

// instanceSpec is the per-challenge provisioning spec (spec 5.8).
type instanceSpec struct {
	Image string            `json:"image"`
	Port  int               `json:"port"`
	TTL   string            `json:"ttl"`
	Env   map[string]string `json:"env"`
}

func parseInstanceSpec(raw json.RawMessage) (instanceSpec, bool) {
	var sp instanceSpec
	if len(raw) == 0 || string(raw) == "null" {
		return sp, false
	}
	if json.Unmarshal(raw, &sp) != nil {
		return sp, false
	}
	return sp, true
}

func (sp instanceSpec) usesTeamFlag() bool {
	return sp.Env != nil && sp.Env["FLAG"] == "{{team_flag}}"
}

// provisionMock stands in for a container provisioner. A real Docker/K8s
// provisioner (mounting the host socket, per-instance network namespace) is the
// documented opt-in; the mock keeps the lifecycle contract testable and the
// server free of container privileges by default (spec 5.8, ADR-0009).
func provisionMock(sp instanceSpec) (host string, port int) {
	host = "10.13.37." + strconv.Itoa(2+rand.Intn(250)) //nolint:gosec // mock provisioner: non-crypto host assignment
	port = sp.Port
	if port == 0 {
		port = 30000 + rand.Intn(20000) //nolint:gosec // mock provisioner: non-crypto port assignment
	}
	return host, port
}

func (s *Server) handleSetInstanceSpec(w http.ResponseWriter, r *http.Request) {
	var in struct {
		InstanceSpec json.RawMessage `json:"instance_spec"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	err := s.store.SetInstanceSpec(r.Context(), chi.URLParam(r, "ecID"), in.InstanceSpec)
	s.respondUpdate(w, err)
}

func (s *Server) instanceContext(w http.ResponseWriter, r *http.Request) (eventID, ecID, teamID string, sp instanceSpec, ok bool) {
	u, _ := userFrom(r)
	eventID = chi.URLParam(r, "eventID")
	ecID = chi.URLParam(r, "ecID")
	tid, err := s.store.PlayerTeamForEvent(r.Context(), eventID, u.ID)
	if errors.Is(err, store.ErrNotFound) {
		s.writeProblem(w, http.StatusForbidden, "Forbidden", "your team is not registered for this event")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "membership lookup failed")
		return
	}
	raw, err := s.store.ChallengeInstanceSpec(r.Context(), ecID)
	if errors.Is(err, store.ErrNotFound) {
		s.writeProblem(w, http.StatusNotFound, "Not found", "challenge not found")
		return
	}
	spec, has := parseInstanceSpec(raw)
	if !has {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", "this challenge has no instance")
		return
	}
	return eventID, ecID, tid, spec, true
}

func (s *Server) instanceResponse(eventID, ecID, teamID string, sp instanceSpec, inst store.Instance) map[string]any {
	resp := map[string]any{"instance": inst}
	if sp.usesTeamFlag() {
		// Demo/mock: the per-team flag is surfaced so it can be exercised end to
		// end. A real deployment injects it as env FLAG inside the container only.
		resp["flag"] = flagpkg.TeamFlag(s.cfg.FlagSecret, eventID, teamID, ecID)
	}
	return resp
}

func (s *Server) handleCreateInstance(w http.ResponseWriter, r *http.Request) {
	eventID, ecID, teamID, sp, ok := s.instanceContext(w, r)
	if !ok {
		return
	}
	if inst, err := s.store.GetActiveInstance(r.Context(), ecID, teamID); err == nil {
		s.writeJSON(w, http.StatusOK, s.instanceResponse(eventID, ecID, teamID, sp, inst))
		return
	}
	n, _ := s.store.CountRunningInstances(r.Context())
	if n >= s.cfg.InstanceMax {
		s.writeProblem(w, http.StatusTooManyRequests, "Too many requests", "global instance limit reached")
		return
	}
	host, port := provisionMock(sp)
	ttl := s.cfg.InstanceTTL
	inst, err := s.store.CreateInstance(r.Context(), ecID, teamID, host, port, time.Now().Add(ttl))
	if errors.Is(err, store.ErrConflict) {
		existing, _ := s.store.GetActiveInstance(r.Context(), ecID, teamID)
		s.writeJSON(w, http.StatusOK, s.instanceResponse(eventID, ecID, teamID, sp, existing))
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "provision failed")
		return
	}
	s.writeJSON(w, http.StatusCreated, s.instanceResponse(eventID, ecID, teamID, sp, inst))
}

func (s *Server) handleGetInstance(w http.ResponseWriter, r *http.Request) {
	eventID, ecID, teamID, sp, ok := s.instanceContext(w, r)
	if !ok {
		return
	}
	inst, err := s.store.GetActiveInstance(r.Context(), ecID, teamID)
	if errors.Is(err, store.ErrNotFound) {
		s.writeProblem(w, http.StatusNotFound, "Not found", "no running instance")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "lookup failed")
		return
	}
	s.writeJSON(w, http.StatusOK, s.instanceResponse(eventID, ecID, teamID, sp, inst))
}

func (s *Server) handleExtendInstance(w http.ResponseWriter, r *http.Request) {
	eventID, ecID, teamID, sp, ok := s.instanceContext(w, r)
	if !ok {
		return
	}
	inst, err := s.store.ExtendInstance(r.Context(), ecID, teamID, s.cfg.InstanceTTL, 2)
	if errors.Is(err, store.ErrNotFound) {
		s.writeProblem(w, http.StatusConflict, "Conflict", "no running instance or extend limit reached")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "extend failed")
		return
	}
	s.writeJSON(w, http.StatusOK, s.instanceResponse(eventID, ecID, teamID, sp, inst))
}

func (s *Server) handleDestroyInstance(w http.ResponseWriter, r *http.Request) {
	_, ecID, teamID, _, ok := s.instanceContext(w, r)
	if !ok {
		return
	}
	destroyed, err := s.store.DestroyInstance(r.Context(), ecID, teamID)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "destroy failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"destroyed": destroyed})
}
