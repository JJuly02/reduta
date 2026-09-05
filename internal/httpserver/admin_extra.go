package httpserver

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/JJuly02/reduta/internal/store"
)

func (s *Server) handleCreateNotification(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	var in struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	if strings.TrimSpace(in.Title) == "" {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "title is required")
		return
	}
	n, err := s.store.CreateNotification(r.Context(), eventID, strings.TrimSpace(in.Title), in.Content)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "create failed")
		return
	}
	s.publish(eventID, "notification", map[string]string{"title": n.Title, "content": n.Content})
	s.writeJSON(w, http.StatusCreated, n)
}

func (s *Server) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	limit := 50
	items, err := s.store.ListNotifications(r.Context(), chi.URLParam(r, "eventID"), limit)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "list failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"notifications": items})
}

func (s *Server) handleListSubmissions(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	items, err := s.store.ListSubmissions(r.Context(), chi.URLParam(r, "eventID"), limit)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "list failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"submissions": items})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	st, err := s.store.Stats(r.Context(), chi.URLParam(r, "eventID"))
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "stats failed")
		return
	}
	_ = store.EventStats{}
	s.writeJSON(w, http.StatusOK, st)
}
