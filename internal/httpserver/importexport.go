package httpserver

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	flagpkg "github.com/JJuly02/reduta/internal/core/flags"
	"github.com/JJuly02/reduta/internal/store"
)

const (
	maxFileBytes   = 25 << 20  // 25 MiB per attached file
	maxImportBytes = 128 << 20 // 128 MiB import body (files are base64 in JSON)
)

type ioFlag struct {
	Value         string `json:"value,omitempty"` // plaintext (import)
	Hash          string `json:"hash,omitempty"`  // hex sha256 (export/round-trip)
	CaseSensitive *bool  `json:"case_sensitive,omitempty"`
}

// ioFile is one attached file in the import/export JSON. On import supply Name and
// Data (base64); ContentType is optional. On export all fields are populated so a
// single JSON round-trips the whole challenge including its files.
type ioFile struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type,omitempty"`
	Size        int64  `json:"size,omitempty"`
	SHA256      string `json:"sha256,omitempty"` // hex
	Data        string `json:"data,omitempty"`   // base64 of the file bytes
}

type ioChallenge struct {
	Title         string          `json:"title"`
	Category      string          `json:"category"`
	DescriptionMD string          `json:"description_md"`
	State         string          `json:"state"`
	Scoring       json.RawMessage `json:"scoring"`
	Tags          []string        `json:"tags"`
	Flags         []ioFlag        `json:"flags"`
	Files         []ioFile        `json:"files,omitempty"`
}

// toStoreFile decodes a base64 file, enforces the per-file cap, and computes its
// digest and size.
func (f ioFile) toStoreFile() (store.ChallengeFileInput, error) {
	name := strings.TrimSpace(f.Name)
	if name == "" {
		return store.ChallengeFileInput{}, errors.New("file with empty name")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(f.Data))
	if err != nil {
		if raw, err = base64.RawStdEncoding.DecodeString(strings.TrimSpace(f.Data)); err != nil {
			return store.ChallengeFileInput{}, fmt.Errorf("file %q: invalid base64", name)
		}
	}
	if int64(len(raw)) > maxFileBytes {
		return store.ChallengeFileInput{}, fmt.Errorf("file %q exceeds the %d MiB limit", name, maxFileBytes>>20)
	}
	ct := strings.TrimSpace(f.ContentType)
	if ct == "" {
		ct = "application/octet-stream"
	}
	sum := sha256.Sum256(raw)
	return store.ChallengeFileInput{Name: name, ContentType: ct, Data: raw, SHA256: sum[:], Size: int64(len(raw))}, nil
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
	// ?files=meta emits file metadata without the (potentially large) bytes.
	includeData := r.URL.Query().Get("files") != "meta"
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
		var files []ioFile
		if metas, err := s.store.ListChallengeFiles(r.Context(), c.ID); err == nil {
			for _, m := range metas {
				of := ioFile{Name: m.Name, ContentType: m.ContentType, Size: m.Size, SHA256: hex.EncodeToString(m.SHA256)}
				if includeData {
					if content, err := s.store.GetChallengeFile(r.Context(), m.ID); err == nil {
						of.Data = base64.StdEncoding.EncodeToString(content.Data)
					}
				}
				files = append(files, of)
			}
		}
		out = append(out, ioChallenge{
			Title: c.Title, Category: c.Category, DescriptionMD: c.DescriptionMD,
			State: c.State, Scoring: c.Scoring, Tags: tags, Flags: flags, Files: files,
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
		// Event and Blocks are accepted (and ignored) so a full export re-imports
		// cleanly; only the challenges array is applied.
		var in struct {
			DryRun     bool            `json:"dry_run"`
			Event      json.RawMessage `json:"event,omitempty"`
			Blocks     json.RawMessage `json:"blocks,omitempty"`
			Challenges []ioChallenge   `json:"challenges"`
		}
		if err := decodeJSONMax(r, &in, maxImportBytes); err != nil {
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
		var files []store.ChallengeFileInput
		for _, inf := range ch.Files {
			sf, ferr := inf.toStoreFile()
			if ferr != nil {
				problems = append(problems, ch.Title+": "+ferr.Error())
				continue
			}
			files = append(files, sf)
		}
		existing, err := s.store.FindECByTitle(r.Context(), eventID, ch.Title)
		switch {
		case errors.Is(err, store.ErrNotFound):
			created++
			if !dryRun {
				newID, cerr := s.store.CreateChallengeWithFlags(r.Context(), eventID, ch.Title, ch.Category, ch.DescriptionMD, ch.Scoring, ch.State, ch.Tags, flags)
				if cerr != nil {
					problems = append(problems, ch.Title+": "+cerr.Error())
				} else if len(files) > 0 {
					if ferr := s.store.ReplaceChallengeFiles(r.Context(), newID, files); ferr != nil {
						problems = append(problems, ch.Title+": files: "+ferr.Error())
					}
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
				if len(files) > 0 {
					if ferr := s.store.ReplaceChallengeFiles(r.Context(), existing.ID, files); ferr != nil {
						problems = append(problems, ch.Title+": files: "+ferr.Error())
					}
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
