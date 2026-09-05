package httpserver

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/JJuly02/reduta/internal/auth"
	"github.com/JJuly02/reduta/internal/store"
)

type pluginCtxKey int

const pluginKey pluginCtxKey = 0

func pluginFrom(r *http.Request) (*store.Plugin, bool) {
	p, ok := r.Context().Value(pluginKey).(*store.Plugin)
	return p, ok
}

func (s *Server) pluginAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			s.writeProblem(w, http.StatusUnauthorized, "Unauthorized", "plugin bearer token required")
			return
		}
		tok := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
		p, err := s.store.PluginByToken(r.Context(), auth.HashToken(tok))
		if err != nil {
			s.writeProblem(w, http.StatusUnauthorized, "Unauthorized", "invalid plugin token")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), pluginKey, &p)))
	})
}

// ---- admin: registration ----

func (s *Server) handleRegisterPlugin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID           string          `json:"id"`
		Name         string          `json:"name"`
		BaseURL      string          `json:"base_url"`
		Capabilities json.RawMessage `json:"capabilities"`
		Events       json.RawMessage `json:"events"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	if strings.TrimSpace(in.ID) == "" || strings.TrimSpace(in.Name) == "" {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "id and name are required")
		return
	}
	rnd, _ := auth.RandomToken(24)
	token := "plg_" + rnd
	secret, _ := auth.RandomToken(24)
	if err := s.store.RegisterPlugin(r.Context(), in.ID, in.Name, in.BaseURL, secret, auth.HashToken(token), in.Capabilities, in.Events); err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "register failed")
		return
	}
	// token and secret are shown once at registration
	s.writeJSON(w, http.StatusCreated, map[string]string{"id": in.ID, "token": token, "secret": secret})
}

func (s *Server) handleListPlugins(w http.ResponseWriter, r *http.Request) {
	plugins, err := s.store.ListPlugins(r.Context())
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "list failed")
		return
	}
	if plugins == nil {
		plugins = []store.Plugin{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"plugins": plugins})
}

// ---- plugin API (Bearer plg_) ----

func (s *Server) handlePluginAward(w http.ResponseWriter, r *http.Request) {
	p, _ := pluginFrom(r)
	var in struct {
		EventID string          `json:"event_id"`
		TeamID  string          `json:"team_id"`
		Points  int             `json:"points"`
		RefID   string          `json:"ref_id"`
		Reason  string          `json:"reason"`
		Meta    json.RawMessage `json:"meta"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	if in.EventID == "" || in.TeamID == "" || in.RefID == "" {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "event_id, team_id and ref_id are required")
		return
	}
	source := "plugin:" + p.ID
	applied, err := s.store.PluginAward(r.Context(), in.EventID, in.TeamID, source, in.RefID, in.Points, in.Meta)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "award failed")
		return
	}
	if applied {
		s.scoreChanged(in.EventID)
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"applied": applied, "ref_id": in.RefID})
}

func (s *Server) handleDeletePluginAward(w http.ResponseWriter, r *http.Request) {
	p, _ := pluginFrom(r)
	eventID := r.URL.Query().Get("event_id")
	refID := chi.URLParam(r, "refID")
	if eventID == "" {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "event_id query is required")
		return
	}
	removed, err := s.store.DeletePluginAward(r.Context(), eventID, "plugin:"+p.ID, refID)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "delete failed")
		return
	}
	if removed {
		s.scoreChanged(eventID)
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"removed": removed})
}

func (s *Server) handlePluginTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := s.store.ListTeams(r.Context(), chi.URLParam(r, "eventID"))
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "list failed")
		return
	}
	if teams == nil {
		teams = []store.Team{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"teams": teams})
}

func (s *Server) handlePluginAnnounce(w http.ResponseWriter, r *http.Request) {
	var in struct {
		EventID string `json:"event_id"`
		Text    string `json:"text"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	if in.EventID == "" || in.Text == "" {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "event_id and text are required")
		return
	}
	p, _ := pluginFrom(r)
	s.publish(in.EventID, "announcement", map[string]string{"from": p.ID, "text": in.Text})
	s.writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}

func (s *Server) handlePluginConfig(w http.ResponseWriter, r *http.Request) {
	p, _ := pluginFrom(r)
	s.writeJSON(w, http.StatusOK, map[string]any{"plugin": p.ID, "config": map[string]any{}})
}

// ---- outbound webhooks (core -> plugin) ----

func subscribed(eventsJSON json.RawMessage, typ string) bool {
	var evs []string
	if json.Unmarshal(eventsJSON, &evs) != nil {
		return false
	}
	for _, e := range evs {
		if e == typ {
			return true
		}
	}
	return false
}

// deliverWebhook signs and POSTs an event to every subscribed plugin (spec 7.2).
func (s *Server) deliverWebhook(eventType string, payload any) {
	plugins, err := s.store.ListPlugins(context.Background())
	if err != nil {
		return
	}
	body, _ := json.Marshal(payload)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	for _, p := range plugins {
		if p.BaseURL == "" || !subscribed(p.Events, eventType) {
			continue
		}
		go func(p store.Plugin) {
			mac := hmac.New(sha256.New, []byte(p.Secret))
			mac.Write([]byte(ts + "." + string(body)))
			sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
			req, err := http.NewRequest(http.MethodPost, strings.TrimRight(p.BaseURL, "/")+"/hooks", bytes.NewReader(body))
			if err != nil {
				return
			}
			delivery, _ := auth.RandomToken(12)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Reduta-Event", eventType)
			req.Header.Set("X-Reduta-Delivery", delivery)
			req.Header.Set("X-Reduta-Timestamp", ts)
			req.Header.Set("X-Reduta-Signature", sig)
			cl := &http.Client{Timeout: 2 * time.Second}
			if resp, err := cl.Do(req); err == nil {
				resp.Body.Close()
			} else {
				s.log.Debug().Err(err).Str("plugin", p.ID).Str("event", eventType).Msg("webhook delivery failed")
			}
		}(p)
	}
}

// tickLoop emits a tick.minute webhook every minute (drives KotH scoring).
func (s *Server) tickLoop(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			s.deliverWebhook("tick.minute", map[string]any{"ts": now.UTC().Format(time.RFC3339)})
		}
	}
}

