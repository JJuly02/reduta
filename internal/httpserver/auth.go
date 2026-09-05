package httpserver

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/JJuly02/reduta/internal/auth"
	"github.com/JJuly02/reduta/internal/store"
)

func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, u store.User) error {
	tok, err := auth.RandomToken(32)
	if err != nil {
		return err
	}
	if err := s.store.CreateSession(r.Context(), auth.HashToken(tok), u.ID, time.Now().Add(s.cfg.SessionTTL)); err != nil {
		return err
	}
	http.SetCookie(w, auth.NewSessionCookie(tok, s.cfg.Env != "dev", s.cfg.SessionTTL))
	return nil
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		Password    string `json:"password"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	if in.Email == "" || in.DisplayName == "" || len(in.Password) < 8 {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "email, display_name and password (min 8) are required")
		return
	}
	orgID, err := s.store.DefaultOrgID(r.Context())
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "no default org")
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "hash failed")
		return
	}
	u, err := s.store.CreateUser(r.Context(), orgID, in.Email, in.DisplayName, hash, "player")
	if errors.Is(err, store.ErrConflict) {
		s.writeProblem(w, http.StatusConflict, "Conflict", "email already registered")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "create user failed")
		return
	}
	if err := s.issueSession(w, r, u); err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "session failed")
		return
	}
	s.writeJSON(w, http.StatusCreated, u)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	u, err := s.store.GetUserByEmail(r.Context(), in.Email)
	if err != nil {
		s.writeProblem(w, http.StatusUnauthorized, "Unauthorized", "invalid credentials")
		return
	}
	ok, err := auth.VerifyPassword(in.Password, u.PasswordHash)
	if err != nil || !ok {
		s.writeProblem(w, http.StatusUnauthorized, "Unauthorized", "invalid credentials")
		return
	}
	if err := s.issueSession(w, r, u); err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "session failed")
		return
	}
	s.writeJSON(w, http.StatusOK, u)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.SessionCookieName); err == nil && c.Value != "" {
		_ = s.store.DeleteSession(r.Context(), auth.HashToken(c.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: auth.SessionCookieName, Value: "", Path: "/", MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u, ok := userFrom(r)
	if !ok {
		s.writeProblem(w, http.StatusUnauthorized, "Unauthorized", "authentication required")
		return
	}
	s.writeJSON(w, http.StatusOK, u)
}
