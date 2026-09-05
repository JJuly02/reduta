package httpserver

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/JJuly02/reduta/internal/store"
)

// handleDownloadFile streams one file attached to a challenge. It is readable by
// anyone who can see the challenge: the challenge must belong to the event and be
// published (admins may fetch draft/hidden), matching handleGetChallenge.
func (s *Server) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r)
	isAdmin := u != nil && (u.Role == "owner" || u.Role == "admin")
	eventID := chi.URLParam(r, "eventID")
	ecID := chi.URLParam(r, "ecID")
	fileID := chi.URLParam(r, "fileID")

	c, err := s.store.GetChallenge(r.Context(), ecID)
	if errors.Is(err, store.ErrNotFound) || (err == nil && c.EventID != eventID) {
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

	f, err := s.store.GetChallengeFile(r.Context(), fileID)
	if errors.Is(err, store.ErrNotFound) || (err == nil && f.ECID != ecID) {
		s.writeProblem(w, http.StatusNotFound, "Not found", "file not found")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "get failed")
		return
	}

	w.Header().Set("Content-Type", f.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(f.Size, 10))
	w.Header().Set("Content-Disposition", `attachment; filename="`+sanitizeFilename(f.Name)+`"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(f.Data)
}

// sanitizeFilename keeps a Content-Disposition filename to a safe basename: no
// path separators, quotes, or control characters that could break the header.
func sanitizeFilename(name string) string {
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	name = strings.Map(func(rr rune) rune {
		if rr < 0x20 || rr == '"' {
			return '_'
		}
		return rr
	}, name)
	if name == "" {
		return "file"
	}
	return name
}
