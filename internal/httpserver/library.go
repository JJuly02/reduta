package httpserver

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	flagpkg "github.com/JJuly02/reduta/internal/core/flags"
	"github.com/JJuly02/reduta/internal/store"
)

type flagInput struct {
	Value         string `json:"value"`
	CaseSensitive *bool  `json:"case_sensitive"`
}

// hashedFlagsJSON turns plaintext flags into the stored [{hash,case_sensitive}] shape.
func hashedFlagsJSON(in []flagInput) json.RawMessage {
	out := make([]store.RevisionFlag, 0, len(in))
	for _, f := range in {
		if strings.TrimSpace(f.Value) == "" {
			continue
		}
		cs := true
		if f.CaseSensitive != nil {
			cs = *f.CaseSensitive
		}
		out = append(out, store.RevisionFlag{Hash: hex.EncodeToString(flagpkg.Hash(f.Value, cs)), CaseSensitive: cs})
	}
	b, _ := json.Marshal(out)
	return b
}

func (s *Server) handleCreateLibraryChallenge(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r)
	var in struct {
		Slug          string          `json:"slug"`
		Title         string          `json:"title"`
		Category      string          `json:"category"`
		Difficulty    *int            `json:"difficulty"`
		Author        *string         `json:"author"`
		Tags          []string        `json:"tags"`
		DescriptionMD string          `json:"description_md"`
		Scoring       json.RawMessage `json:"scoring"`
		ConnectionTpl *string         `json:"connection_tpl"`
		Flags         []flagInput     `json:"flags"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	if strings.TrimSpace(in.Slug) == "" || strings.TrimSpace(in.Title) == "" {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "slug and title are required")
		return
	}
	if len(in.Flags) == 0 {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "at least one flag is required")
		return
	}
	if in.Category == "" {
		in.Category = "misc"
	}
	c, err := s.store.CreateLibraryChallenge(r.Context(), u.OrgID, strings.TrimSpace(in.Slug), strings.TrimSpace(in.Title),
		in.Category, in.Difficulty, in.Author, in.Tags, in.DescriptionMD, in.Scoring, hashedFlagsJSON(in.Flags), in.ConnectionTpl, u.ID)
	if errors.Is(err, store.ErrConflict) {
		s.writeProblem(w, http.StatusConflict, "Conflict", "slug already exists in library")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "create failed")
		return
	}
	s.writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleListLibrary(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r)
	list, err := s.store.ListLibrary(r.Context(), u.OrgID, r.URL.Query().Get("tag"))
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "list failed")
		return
	}
	if list == nil {
		list = []store.Challenge{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"challenges": list})
}

func (s *Server) handleGetLibraryChallenge(w http.ResponseWriter, r *http.Request) {
	cid := chi.URLParam(r, "cid")
	c, err := s.store.GetLibraryChallenge(r.Context(), cid)
	if errors.Is(err, store.ErrNotFound) {
		s.writeProblem(w, http.StatusNotFound, "Not found", "challenge not found")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "get failed")
		return
	}
	rev, err := s.store.GetRevision(r.Context(), cid, c.CurrentRev)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "revision failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"challenge": c, "revision": rev})
}

func (s *Server) handleListRevisions(w http.ResponseWriter, r *http.Request) {
	revs, err := s.store.ListRevisions(r.Context(), chi.URLParam(r, "cid"))
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "list failed")
		return
	}
	if revs == nil {
		revs = []store.Revision{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"revisions": revs})
}

func (s *Server) handleNewRevision(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r)
	cid := chi.URLParam(r, "cid")
	var in struct {
		DescriptionMD string          `json:"description_md"`
		Scoring       json.RawMessage `json:"scoring"`
		ConnectionTpl *string         `json:"connection_tpl"`
		Flags         []flagInput     `json:"flags"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	rev, err := s.store.NewRevision(r.Context(), cid, in.DescriptionMD, in.Scoring, hashedFlagsJSON(in.Flags), in.ConnectionTpl, u.ID)
	if errors.Is(err, store.ErrNotFound) {
		s.writeProblem(w, http.StatusNotFound, "Not found", "challenge not found")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "new revision failed")
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{"rev": rev})
}

func (s *Server) handleCloneChallenge(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r)
	var in struct {
		Slug  string `json:"slug"`
		Title string `json:"title"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	if strings.TrimSpace(in.Slug) == "" {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "slug is required")
		return
	}
	c, err := s.store.CloneChallenge(r.Context(), chi.URLParam(r, "cid"), strings.TrimSpace(in.Slug), strings.TrimSpace(in.Title), u.ID)
	if errors.Is(err, store.ErrNotFound) {
		s.writeProblem(w, http.StatusNotFound, "Not found", "source challenge not found")
		return
	}
	if errors.Is(err, store.ErrConflict) {
		s.writeProblem(w, http.StatusConflict, "Conflict", "target slug already exists")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "clone failed")
		return
	}
	s.writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleEmbed(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	var in struct {
		ChallengeID string `json:"challenge_id"`
		Rev         int    `json:"rev"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	if in.ChallengeID == "" {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "challenge_id is required")
		return
	}
	c, err := s.store.EmbedFromLibrary(r.Context(), eventID, in.ChallengeID, in.Rev)
	if errors.Is(err, store.ErrNotFound) {
		s.writeProblem(w, http.StatusNotFound, "Not found", "challenge or revision not found")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "embed failed")
		return
	}
	s.writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleCreateBlock(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	var in struct {
		Name       string          `json:"name"`
		Position   int             `json:"position"`
		Color      *string         `json:"color"`
		Schedule   json.RawMessage `json:"schedule"`
		UnlockRule json.RawMessage `json:"unlock_rule"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "name is required")
		return
	}
	b, err := s.store.CreateBlock(r.Context(), eventID, strings.TrimSpace(in.Name), in.Position, in.Color, in.Schedule, in.UnlockRule)
	if errors.Is(err, store.ErrConflict) {
		s.writeProblem(w, http.StatusConflict, "Conflict", "block name already exists")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "create block failed")
		return
	}
	s.writeJSON(w, http.StatusCreated, b)
}

func (s *Server) handleListBlocks(w http.ResponseWriter, r *http.Request) {
	blocks, err := s.store.ListBlocks(r.Context(), chi.URLParam(r, "eventID"))
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "list failed")
		return
	}
	if blocks == nil {
		blocks = []store.Block{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"blocks": blocks})
}

func (s *Server) handleBulk(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r)
	eventID := chi.URLParam(r, "eventID")
	var in struct {
		Selector struct {
			Mode       string         `json:"mode"`
			IDs        []string       `json:"ids"`
			Filter     map[string]any `json:"filter"`
			ExcludeIDs []string       `json:"exclude_ids"`
		} `json:"selector"`
		Action struct {
			Type   string         `json:"type"`
			Params map[string]any `json:"params"`
		} `json:"action"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	if !oneOf(in.Action.Type, "publish", "hide", "archive", "set_state", "assign_block", "add_tags", "remove_tags", "set_schedule", "delete") {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "unknown action")
		return
	}
	ids, err := s.store.ResolveSelector(r.Context(), eventID, in.Selector.Mode, in.Selector.IDs, in.Selector.Filter, in.Selector.ExcludeIDs)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "selector failed")
		return
	}
	jobID, affected, err := s.store.BulkApply(r.Context(), eventID, u.ID, in.Action.Type, in.Action.Params, ids)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "bulk failed")
		return
	}
	s.publish(eventID, "challenges.changed", nil)
	// M1-scale: applied synchronously (spec 5.5 async threshold is a later optimization).
	s.writeJSON(w, http.StatusOK, map[string]any{"job_id": jobID, "affected": affected})
}

func (s *Server) handleBulkUndo(w http.ResponseWriter, r *http.Request) {
	n, err := s.store.BulkUndo(r.Context(), chi.URLParam(r, "jobID"))
	if errors.Is(err, store.ErrNotFound) {
		s.writeProblem(w, http.StatusNotFound, "Not found", "job not found")
		return
	}
	if errors.Is(err, store.ErrConflict) {
		s.writeProblem(w, http.StatusConflict, "Conflict", "nothing to undo")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "undo failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"restored": n})
}

func (s *Server) handleCreateSavedView(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r)
	eventID := chi.URLParam(r, "eventID")
	var in struct {
		Name   string          `json:"name"`
		Filter json.RawMessage `json:"filter"`
	}
	if err := decodeJSON(r, &in); err != nil {
		s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		s.writeProblem(w, http.StatusUnprocessableEntity, "Validation failed", "name is required")
		return
	}
	id, err := s.store.CreateSavedView(r.Context(), eventID, u.ID, strings.TrimSpace(in.Name), in.Filter)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "save failed")
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) handleListSavedViews(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r)
	views, err := s.store.ListSavedViews(r.Context(), chi.URLParam(r, "eventID"), u.ID)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "list failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"saved_views": views})
}
