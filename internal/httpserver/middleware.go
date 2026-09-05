package httpserver

import (
	"context"
	"net/http"

	"github.com/JJuly02/reduta/internal/auth"
	"github.com/JJuly02/reduta/internal/store"
)

type ctxKey int

const userKey ctxKey = iota

func (s *Server) sessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(auth.SessionCookieName); err == nil && c.Value != "" {
			if u, err := s.store.SessionUser(r.Context(), auth.HashToken(c.Value)); err == nil {
				r = r.WithContext(context.WithValue(r.Context(), userKey, &u))
			}
		}
		next.ServeHTTP(w, r)
	})
}

func userFrom(r *http.Request) (*store.User, bool) {
	u, ok := r.Context().Value(userKey).(*store.User)
	return u, ok
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := userFrom(r); !ok {
			s.writeProblem(w, http.StatusUnauthorized, "Unauthorized", "authentication required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := userFrom(r)
		if !ok || (u.Role != "owner" && u.Role != "admin") {
			s.writeProblem(w, http.StatusForbidden, "Forbidden", "admin role required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
