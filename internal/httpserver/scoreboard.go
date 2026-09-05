package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/JJuly02/reduta/internal/store"
)

func (s *Server) handleScoreboard(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	cacheKey := "sb:" + eventID

	// Redis cache (2s), per spec 5.7. Cache only the default page; invalidated on writes.
	useCache := s.redis != nil && limit == 100
	if useCache {
		if v, err := s.redis.Get(r.Context(), cacheKey).Bytes(); err == nil && len(v) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "hit")
			_, _ = w.Write(v)
			return
		}
	}

	rows, err := s.store.Scoreboard(r.Context(), eventID, limit)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "scoreboard failed")
		return
	}
	if rows == nil {
		rows = []store.ScoreRow{}
	}
	body, _ := json.Marshal(map[string]any{"event_id": eventID, "entries": rows})
	if useCache {
		_ = s.redis.Set(r.Context(), cacheKey, body, 2*time.Second).Err()
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (s *Server) handleScoreboardSeries(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	top := 10
	if v := r.URL.Query().Get("top"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 25 {
			top = n
		}
	}
	series, err := s.store.ScoreboardSeries(r.Context(), eventID, top)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "series failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"event_id": eventID, "series": series})
}
