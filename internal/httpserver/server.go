// Package httpserver builds the core HTTP API (chi). Handlers are split by
// resource across the files in this package; Server carries shared deps.
package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/JJuly02/reduta/internal/config"
	"github.com/JJuly02/reduta/internal/ratelimit"
	"github.com/JJuly02/reduta/internal/store"
	"github.com/JJuly02/reduta/internal/ws"
)

type Server struct {
	cfg   config.Config
	store *store.Store
	log   zerolog.Logger
	rl    *ratelimit.Limiter
	hub   *ws.Hub
	redis *redis.Client
}

func New(cfg config.Config, st *store.Store, log zerolog.Logger) *Server {
	s := &Server{
		cfg:   cfg,
		store: st,
		log:   log,
		rl:    ratelimit.New(cfg.SubmitRatePerMin, time.Minute),
		hub:   ws.NewHub(),
	}
	if cfg.RedisAddr != "" {
		s.redis = redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
		go s.subscribe(context.Background())
	}
	go s.tickLoop(context.Background())
	return s
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(s.sessionMiddleware)

	r.Get("/", s.handleIndex)
	r.Get("/healthz", s.handleHealth)
	r.Get("/ws", s.handleWS) // /ws?event={id} — no Timeout middleware (needs Hijacker)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Timeout(30 * time.Second))
		r.Get("/health", s.handleHealth)

		r.Post("/auth/register", s.handleRegister)
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/logout", s.handleLogout)
		r.Get("/auth/me", s.handleMe)

		// public reads
		r.Get("/events", s.handleListEvents)
		r.Get("/events/{eventID}", s.handleGetEvent)
		r.Get("/events/{eventID}/scoreboard", s.handleScoreboard)
		r.Get("/events/{eventID}/scoreboard/series", s.handleScoreboardSeries)
		r.Get("/events/{eventID}/challenges", s.handleListChallenges)
		r.Get("/events/{eventID}/challenges/{ecID}", s.handleGetChallenge)
		r.Get("/events/{eventID}/blocks", s.handleListBlocks)
		r.Get("/events/{eventID}/notifications", s.handleListNotifications)

		// authenticated players
		r.Group(func(r chi.Router) {
			r.Use(s.requireAuth)
			r.Post("/teams", s.handleCreateTeam)
			r.Post("/teams/join", s.handleJoinTeam)
			r.Get("/me/team", s.handleMyTeam)
			r.Get("/events/{eventID}/teams", s.handleListTeams)
			r.Get("/events/{eventID}/me", s.handleMyStatus)
			r.Post("/events/{eventID}/challenges/{ecID}/submit", s.handleSubmit)
			r.Get("/events/{eventID}/challenges/{ecID}/attempts", s.handleMyAttempts)
			// M8: per-team instances
			r.Post("/events/{eventID}/challenges/{ecID}/instance", s.handleCreateInstance)
			r.Get("/events/{eventID}/challenges/{ecID}/instance", s.handleGetInstance)
			r.Post("/events/{eventID}/challenges/{ecID}/instance/extend", s.handleExtendInstance)
			r.Delete("/events/{eventID}/challenges/{ecID}/instance", s.handleDestroyInstance)
		})

		// admins (owner|admin)
		r.Group(func(r chi.Router) {
			r.Use(s.requireAuth)
			r.Use(s.requireAdmin)
			r.Post("/events", s.handleCreateEvent)
			r.Patch("/events/{eventID}/state", s.handleSetEventState)
			r.Post("/events/{eventID}/challenges", s.handleCreateChallenge)
			r.Patch("/events/{eventID}/challenges/{ecID}/state", s.handleSetChallengeState)

			// M2: challenge library, revisions, cloning
			r.Post("/challenges", s.handleCreateLibraryChallenge)
			r.Get("/challenges", s.handleListLibrary)
			r.Get("/challenges/{cid}", s.handleGetLibraryChallenge)
			r.Get("/challenges/{cid}/revisions", s.handleListRevisions)
			r.Post("/challenges/{cid}/revisions", s.handleNewRevision)
			r.Post("/challenges/{cid}/clone", s.handleCloneChallenge)
			// M2: blocks, embedding, bulk actions, saved views
			r.Post("/events/{eventID}/blocks", s.handleCreateBlock)
			r.Post("/events/{eventID}/challenges/from-library", s.handleEmbed)
			r.Post("/events/{eventID}/challenges/bulk", s.handleBulk)
			r.Post("/bulk-jobs/{jobID}/undo", s.handleBulkUndo)
			r.Post("/events/{eventID}/saved-views", s.handleCreateSavedView)
			r.Get("/events/{eventID}/saved-views", s.handleListSavedViews)
			// M3: schedules + unlock rules
			r.Patch("/events/{eventID}/challenges/{ecID}/schedule", s.handleSetChallengeSchedule)
			r.Patch("/events/{eventID}/challenges/{ecID}/unlock", s.handleSetChallengeUnlock)
			r.Patch("/blocks/{blockID}/schedule", s.handleSetBlockSchedule)
			r.Patch("/events/{eventID}/challenges/{ecID}/instance-spec", s.handleSetInstanceSpec)
			// M4: import/export
			r.Get("/events/{eventID}/export", s.handleExport)
			r.Post("/events/{eventID}/import", s.handleImport)
			// M7: plugin registration
			r.Post("/plugins", s.handleRegisterPlugin)
			r.Get("/plugins", s.handleListPlugins)
			// global teams + event participation
			r.Get("/teams", s.handleListOrgTeams)
			r.Post("/events/{eventID}/event-teams", s.handleAssignEventTeam)
			r.Delete("/events/{eventID}/event-teams/{teamID}", s.handleUnassignEventTeam)
			// admin extras
			r.Post("/events/{eventID}/notifications", s.handleCreateNotification)
			r.Get("/events/{eventID}/submissions", s.handleListSubmissions)
			r.Get("/events/{eventID}/stats", s.handleStats)
		})

		// M7: plugin API (Bearer plg_ token)
		r.Group(func(r chi.Router) {
			r.Use(s.pluginAuth)
			r.Post("/plugin/v1/awards", s.handlePluginAward)
			r.Delete("/plugin/v1/awards/{refID}", s.handleDeletePluginAward)
			r.Get("/plugin/v1/events/{eventID}/teams", s.handlePluginTeams)
			r.Post("/plugin/v1/announcements", s.handlePluginAnnounce)
			r.Get("/plugin/v1/config", s.handlePluginConfig)
		})
	})
	return r
}

func (s *Server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.store.Ping(ctx); err != nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "degraded", "db": "down"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "db": "ok"})
}

// ---- realtime (M5) ----

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	eventID := r.URL.Query().Get("event")
	if eventID == "" {
		http.Error(w, "event query param required", http.StatusBadRequest)
		return
	}
	_ = s.hub.Serve(r.Context(), w, r, eventID)
}

type wsEnvelope struct {
	EventID string          `json:"event_id"`
	Msg     json.RawMessage `json:"msg"`
}

// subscribe bridges Redis pub/sub into the local hub for cross-instance fan-out.
func (s *Server) subscribe(ctx context.Context) {
	sub := s.redis.Subscribe(ctx, "reduta:events")
	for m := range sub.Channel() {
		var env wsEnvelope
		if json.Unmarshal([]byte(m.Payload), &env) == nil {
			s.hub.Broadcast(env.EventID, env.Msg)
		}
	}
}

// scoreChanged invalidates the scoreboard cache and pushes a live update.
func (s *Server) scoreChanged(eventID string) {
	if s.redis != nil {
		s.redis.Del(context.Background(), "sb:"+eventID)
	}
	s.publish(eventID, "scoreboard", nil)
}

// publish emits a realtime message for an event to all instances via Redis.
func (s *Server) publish(eventID, typ string, data any) {
	if s.redis == nil {
		return
	}
	msg, _ := json.Marshal(map[string]any{"type": typ, "data": data})
	env, _ := json.Marshal(wsEnvelope{EventID: eventID, Msg: msg})
	s.redis.Publish(context.Background(), "reduta:events", env)
}

// ---- shared helpers ----

func (s *Server) writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

type problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// writeProblem emits an RFC 9457 application/problem+json error (spec 6).
func (s *Server) writeProblem(w http.ResponseWriter, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem{Type: "about:blank", Title: title, Status: status, Detail: detail})
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 4<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func parseTimePtr(v *string) (*time.Time, error) {
	if v == nil || *v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *v)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func oneOf(v string, opts ...string) bool {
	for _, o := range opts {
		if v == o {
			return true
		}
	}
	return false
}

const indexHTML = `<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Reduta</title>
<style>
  :root{color-scheme:dark}
  body{margin:0;min-height:100vh;display:grid;place-items:center;
    background:oklch(0.16 0.01 260);color:oklch(0.92 0.02 260);
    font:16px/1.5 ui-sans-serif,Inter,system-ui,sans-serif}
  main{max-width:34rem;padding:2rem}
  h1{font-size:2rem;margin:0 0 .25rem;letter-spacing:-.02em}
  .tag{color:oklch(0.72 0.16 240);font-family:ui-monospace,"JetBrains Mono",monospace}
  a{color:oklch(0.72 0.16 240)}
  code{font-family:ui-monospace,"JetBrains Mono",monospace}
</style></head>
<body><main>
  <h1>Reduta <span class="tag">API</span></h1>
  <p>Core API: auth, events, teams, challenges, submissions, scoreboard, realtime.</p>
  <p>The player &amp; admin UI is served by the <code>web</code> service.</p>
  <p>Health: <a href="/healthz"><code>/healthz</code></a></p>
</main></body></html>`
