package httpserver

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	flagpkg "github.com/JJuly02/reduta/internal/core/flags"
	"github.com/JJuly02/reduta/internal/store"
)

type ioFlag struct {
	Value         string `json:"value,omitempty"` // plaintext (import)
	Hash          string `json:"hash,omitempty"`  // hex sha256 (export/round-trip)
	CaseSensitive *bool  `json:"case_sensitive,omitempty"`
}

type ioChallenge struct {
	Title         string          `json:"title"`
	Category      string          `json:"category"`
	DescriptionMD string          `json:"description_md"`
	State         string          `json:"state"`
	Scoring       json.RawMessage `json:"scoring"`
	Tags          []string        `json:"tags"`
	Flags         []ioFlag        `json:"flags"`
}

func (f ioFlag) toStoreFlag() (store.Flag, bool) {
	cs := true
	if f.CaseSensitive != nil {
		cs = *f.CaseSensitive
	}
	if f.Hash != "" {
		hb, err := hex.DecodeString(f.Hash)
		if err != nil {
			return store.Flag{}, false
		}
		return store.Flag{ValueHash: hb, CaseSensitive: cs}, true
	}
	if f.Value == "" {
		return store.Flag{}, false
	}
	return store.Flag{ValueHash: flagpkg.Hash(f.Value, cs), CaseSensitive: cs}, true
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	ev, err := s.store.GetEvent(r.Context(), eventID)
	if errors.Is(err, store.ErrNotFound) {
		s.writeProblem(w, http.StatusNotFound, "Not found", "event not found")
		return
	}
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "export failed")
		return
	}
	blocks, _ := s.store.ListBlocks(r.Context(), eventID)
	chs, err := s.store.ListChallenges(r.Context(), eventID, false)
	if err != nil {
		s.writeProblem(w, http.StatusInternalServerError, "Internal error", "export failed")
		return
	}
	out := make([]ioChallenge, 0, len(chs))
	for _, c := range chs {
		fs, _ := s.store.ChallengeFlags(r.Context(), c.ID)
		flags := make([]ioFlag, 0, len(fs))
		for _, f := range fs {
			cs := f.CaseSensitive
			flags = append(flags, ioFlag{Hash: hex.EncodeToString(f.ValueHash), CaseSensitive: &cs})
		}
		tags := c.Tags
		if tags == nil {
			tags = []string{}
		}
		out = append(out, ioChallenge{
			Title: c.Title, Category: c.Category, DescriptionMD: c.DescriptionMD,
			State: c.State, Scoring: c.Scoring, Tags: tags, Flags: flags,
		})
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+ev.Slug+`.json"`)
	s.writeJSON(w, http.StatusOK, map[string]any{
		"event":      map[string]string{"slug": ev.Slug, "name": ev.Name},
		"blocks":     blocks,
		"challenges": out,
	})
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	if _, err := s.store.GetEvent(r.Context(), eventID); errors.Is(err, store.ErrNotFound) {
		s.writeProblem(w, http.StatusNotFound, "Not found", "event not found")
		return
	}
	dryRun := r.URL.Query().Get("dry_run") == "true"

	var challenges []ioChallenge
	if r.URL.Query().Get("format") == "ctfd" {
		parsed, err := parseCTFd(r.Body)
		if err != nil {
			s.writeProblem(w, http.StatusBadRequest, "Bad request", "invalid CTFd export: "+err.Error())
			return
		}
		challenges = parsed
	} else {
		var in struct {
			DryRun     bool          `json:"dry_run"`
			Challenges []ioChallenge `json:"challenges"`
		}
		if err := decodeJSON(r, &in); err != nil {
			s.writeProblem(w, http.StatusBadRequest, "Bad request", err.Error())
			return
		}
		challenges = in.Challenges
		dryRun = dryRun || in.DryRun
	}

	created, updated := 0, 0
	problems := []string{}
	for _, ch := range challenges {
		if ch.Title == "" {
			problems = append(problems, "challenge with empty title skipped")
			continue
		}
		if ch.Category == "" {
			ch.Category = "misc"
		}
		if ch.State == "" {
			ch.State = "draft"
		}
		var flags []store.Flag
		for _, f := range ch.Flags {
			if sf, ok := f.toStoreFlag(); ok {
				flags = append(flags, sf)
			}
		}
		existing, err := s.store.FindECByTitle(r.Context(), eventID, ch.Title)
		switch {
		case errors.Is(err, store.ErrNotFound):
			created++
			if !dryRun {
				if _, err := s.store.CreateChallengeWithFlags(r.Context(), eventID, ch.Title, ch.Category, ch.DescriptionMD, ch.Scoring, ch.State, ch.Tags, flags); err != nil {
					problems = append(problems, ch.Title+": "+err.Error())
				}
			}
		case err == nil:
			updated++
			if !dryRun {
				if err := s.store.UpdateChallengeContent(r.Context(), existing.ID, ch.Category, ch.DescriptionMD, ch.Scoring, ch.State, ch.Tags); err != nil {
					problems = append(problems, ch.Title+": "+err.Error())
				}
				if len(flags) > 0 {
					_ = s.store.ReplaceFlags(r.Context(), existing.ID, flags)
				}
			}
		default:
			problems = append(problems, ch.Title+": lookup failed")
		}
	}
	if !dryRun {
		s.publish(eventID, "challenges.changed", nil)
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"dry_run": dryRun,
		"plan":    map[string]int{"created": created, "updated": updated},
		"errors":  problems,
	})
}

// parseCTFd maps a CTFd challenges export into our import shape. Unknown/plugin
// challenge types import as standard; regex flags are skipped with a note.
func parseCTFd(body io.Reader) ([]ioChallenge, error) {
	data, err := io.ReadAll(io.LimitReader(body, 8<<20))
	if err != nil {
		return nil, err
	}
	var doc struct {
		Challenges []struct {
			Name        string            `json:"name"`
			Category    string            `json:"category"`
			Description string            `json:"description"`
			Value       int               `json:"value"`
			State       string            `json:"state"`
			Flags       []json.RawMessage `json:"flags"`
		} `json:"challenges"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	out := make([]ioChallenge, 0, len(doc.Challenges))
	cs := true
	for _, c := range doc.Challenges {
		state := "draft"
		if c.State == "visible" {
			state = "published"
		}
		scoring, _ := json.Marshal(map[string]any{"type": "static", "points": c.Value})
		var flags []ioFlag
		for _, fr := range c.Flags {
			var str string
			if json.Unmarshal(fr, &str) == nil && str != "" {
				flags = append(flags, ioFlag{Value: str, CaseSensitive: &cs})
				continue
			}
			var obj struct {
				Content string `json:"content"`
				Type    string `json:"type"`
			}
			if json.Unmarshal(fr, &obj) == nil && obj.Content != "" && obj.Type != "regex" {
				flags = append(flags, ioFlag{Value: obj.Content, CaseSensitive: &cs})
			}
		}
		out = append(out, ioChallenge{
			Title: c.Name, Category: c.Category, DescriptionMD: c.Description,
			State: state, Scoring: scoring, Flags: flags,
		})
	}
	return out, nil
}
