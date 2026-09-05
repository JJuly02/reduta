package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ---------------- models ----------------

type Challenge struct {
	ID         string     `json:"id"`
	OrgID      string     `json:"org_id"`
	Slug       string     `json:"slug"`
	Title      string     `json:"title"`
	Category   string     `json:"category"`
	Difficulty *int       `json:"difficulty"`
	Author     *string    `json:"author"`
	Tags       []string   `json:"tags"`
	CurrentRev int        `json:"current_rev"`
	ArchivedAt *time.Time `json:"archived_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

type Revision struct {
	ChallengeID   string          `json:"challenge_id"`
	Rev           int             `json:"rev"`
	DescriptionMD string          `json:"description_md"`
	Scoring       json.RawMessage `json:"scoring"`
	Flags         json.RawMessage `json:"-"` // hashed flags; never serialized
	ConnectionTpl *string         `json:"connection_tpl"`
	CreatedAt     time.Time       `json:"created_at"`
}

type Block struct {
	ID         string          `json:"id"`
	EventID    string          `json:"event_id"`
	Name       string          `json:"name"`
	Position   int             `json:"position"`
	Color      *string         `json:"color"`
	Schedule   json.RawMessage `json:"schedule"`
	UnlockRule json.RawMessage `json:"unlock_rule"`
	CreatedAt  time.Time       `json:"created_at"`
}

// RevisionFlag is the stored (hashed) flag shape inside a revision's flags JSON.
type RevisionFlag struct {
	Hash          string `json:"hash"` // hex sha256
	CaseSensitive bool   `json:"case_sensitive"`
}

// ---------------- library ----------------

func (s *Store) CreateLibraryChallenge(ctx context.Context, orgID, slug, title, category string, difficulty *int, author *string, tags []string, descMD string, scoring, flags json.RawMessage, connTpl *string, createdBy string) (Challenge, error) {
	if tags == nil {
		tags = []string{}
	}
	if len(scoring) == 0 {
		scoring = json.RawMessage(`{"type":"static","points":100}`)
	}
	if len(flags) == 0 {
		flags = json.RawMessage(`[]`)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Challenge{}, err
	}
	defer tx.Rollback(ctx)

	var c Challenge
	err = tx.QueryRow(ctx,
		`INSERT INTO challenges (org_id, slug, title, category, difficulty, author, tags)
		 VALUES ($1::uuid,$2,$3,$4,$5,$6,$7)
		 RETURNING id::text, org_id::text, slug, title, category, difficulty, author, tags, current_rev, archived_at, created_at`,
		orgID, slug, title, category, difficulty, author, tags).
		Scan(&c.ID, &c.OrgID, &c.Slug, &c.Title, &c.Category, &c.Difficulty, &c.Author, &c.Tags, &c.CurrentRev, &c.ArchivedAt, &c.CreatedAt)
	if err != nil {
		if isUnique(err) {
			return Challenge{}, ErrConflict
		}
		return Challenge{}, err
	}
	if _, err = tx.Exec(ctx,
		`INSERT INTO challenge_revisions (challenge_id, rev, description_md, scoring, flags, connection_tpl, created_by)
		 VALUES ($1::uuid,1,$2,$3::jsonb,$4::jsonb,$5,$6::uuid)`,
		c.ID, descMD, string(scoring), string(flags), connTpl, nilIfEmpty(createdBy)); err != nil {
		return Challenge{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Challenge{}, err
	}
	return c, nil
}

func scanChallenge(row pgx.Row) (Challenge, error) {
	var c Challenge
	err := row.Scan(&c.ID, &c.OrgID, &c.Slug, &c.Title, &c.Category, &c.Difficulty, &c.Author, &c.Tags, &c.CurrentRev, &c.ArchivedAt, &c.CreatedAt)
	return c, err
}

const challengeCols = `id::text, org_id::text, slug, title, category, difficulty, author, tags, current_rev, archived_at, created_at`

func (s *Store) GetLibraryChallenge(ctx context.Context, id string) (Challenge, error) {
	c, err := scanChallenge(s.pool.QueryRow(ctx, `SELECT `+challengeCols+` FROM challenges WHERE id=$1::uuid`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Challenge{}, ErrNotFound
	}
	return c, err
}

func (s *Store) ListLibrary(ctx context.Context, orgID string, tag string) ([]Challenge, error) {
	q := `SELECT ` + challengeCols + ` FROM challenges WHERE org_id=$1::uuid AND archived_at IS NULL`
	args := []any{orgID}
	if tag != "" {
		args = append(args, []string{tag})
		q += fmt.Sprintf(` AND tags @> $%d`, len(args))
	}
	q += ` ORDER BY created_at DESC`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Challenge
	for rows.Next() {
		c, err := scanChallenge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetRevision(ctx context.Context, challengeID string, rev int) (Revision, error) {
	var r Revision
	var scoring, flags []byte
	err := s.pool.QueryRow(ctx,
		`SELECT challenge_id::text, rev, description_md, scoring, flags, connection_tpl, created_at
		 FROM challenge_revisions WHERE challenge_id=$1::uuid AND rev=$2`, challengeID, rev).
		Scan(&r.ChallengeID, &r.Rev, &r.DescriptionMD, &scoring, &flags, &r.ConnectionTpl, &r.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Revision{}, ErrNotFound
	}
	r.Scoring, r.Flags = scoring, flags
	return r, err
}

func (s *Store) ListRevisions(ctx context.Context, challengeID string) ([]Revision, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT challenge_id::text, rev, description_md, scoring, flags, connection_tpl, created_at
		 FROM challenge_revisions WHERE challenge_id=$1::uuid ORDER BY rev DESC`, challengeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Revision
	for rows.Next() {
		var r Revision
		var scoring, flags []byte
		if err := rows.Scan(&r.ChallengeID, &r.Rev, &r.DescriptionMD, &scoring, &flags, &r.ConnectionTpl, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Scoring, r.Flags = scoring, flags
		out = append(out, r)
	}
	return out, rows.Err()
}

// NewRevision appends a content revision and bumps current_rev.
func (s *Store) NewRevision(ctx context.Context, challengeID, descMD string, scoring, flags json.RawMessage, connTpl *string, createdBy string) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var next int
	if err = tx.QueryRow(ctx, `UPDATE challenges SET current_rev = current_rev + 1 WHERE id=$1::uuid RETURNING current_rev`, challengeID).Scan(&next); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	if len(scoring) == 0 {
		scoring = json.RawMessage(`{"type":"static","points":100}`)
	}
	if len(flags) == 0 {
		flags = json.RawMessage(`[]`)
	}
	if _, err = tx.Exec(ctx,
		`INSERT INTO challenge_revisions (challenge_id, rev, description_md, scoring, flags, connection_tpl, created_by)
		 VALUES ($1::uuid,$2,$3,$4::jsonb,$5::jsonb,$6,$7::uuid)`,
		challengeID, next, descMD, string(scoring), string(flags), connTpl, nilIfEmpty(createdBy)); err != nil {
		return 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}
	return next, nil
}

// CloneChallenge creates a new library challenge from the latest revision of src.
func (s *Store) CloneChallenge(ctx context.Context, srcID, newSlug, newTitle, createdBy string) (Challenge, error) {
	src, err := s.GetLibraryChallenge(ctx, srcID)
	if err != nil {
		return Challenge{}, err
	}
	rev, err := s.GetRevision(ctx, srcID, src.CurrentRev)
	if err != nil {
		return Challenge{}, err
	}
	if newTitle == "" {
		newTitle = src.Title + " (copy)"
	}
	return s.CreateLibraryChallenge(ctx, src.OrgID, newSlug, newTitle, src.Category, src.Difficulty, src.Author, src.Tags, rev.DescriptionMD, rev.Scoring, rev.Flags, rev.ConnectionTpl, createdBy)
}

// EmbedFromLibrary embeds a challenge revision into an event as an event_challenge,
// copying content + flags (pinning the revision so later library edits don't change
// a running event).
func (s *Store) EmbedFromLibrary(ctx context.Context, eventID, challengeID string, rev int) (EventChallenge, error) {
	lib, err := s.GetLibraryChallenge(ctx, challengeID)
	if err != nil {
		return EventChallenge{}, err
	}
	if rev == 0 {
		rev = lib.CurrentRev
	}
	r, err := s.GetRevision(ctx, challengeID, rev)
	if err != nil {
		return EventChallenge{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return EventChallenge{}, err
	}
	defer tx.Rollback(ctx)

	var c EventChallenge
	var raw []byte
	err = tx.QueryRow(ctx,
		`INSERT INTO event_challenges (event_id, title, category, description_md, scoring, state, challenge_id, rev, tags)
		 VALUES ($1::uuid,$2,$3,$4,$5::jsonb,'draft',$6::uuid,$7,$8)
		 RETURNING id::text, event_id::text, title, category, description_md, state, scoring, position, created_at`,
		eventID, lib.Title, lib.Category, r.DescriptionMD, string(r.Scoring), challengeID, rev, lib.Tags).
		Scan(&c.ID, &c.EventID, &c.Title, &c.Category, &c.DescriptionMD, &c.State, &raw, &c.Position, &c.CreatedAt)
	if err != nil {
		return EventChallenge{}, err
	}
	c.Scoring = json.RawMessage(raw)

	var rflags []RevisionFlag
	_ = json.Unmarshal(r.Flags, &rflags)
	for _, f := range rflags {
		hb, err := hexDecode(f.Hash)
		if err != nil {
			continue
		}
		if _, err = tx.Exec(ctx, `INSERT INTO flags (ec_id, value_hash, case_sensitive) VALUES ($1::uuid,$2,$3)`, c.ID, hb, f.CaseSensitive); err != nil {
			return EventChallenge{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return EventChallenge{}, err
	}
	return c, nil
}

// ---------------- blocks ----------------

func (s *Store) CreateBlock(ctx context.Context, eventID, name string, position int, color *string, schedule, unlock json.RawMessage) (Block, error) {
	var b Block
	var sc, ur []byte
	err := s.pool.QueryRow(ctx,
		`INSERT INTO blocks (event_id, name, position, color, schedule, unlock_rule)
		 VALUES ($1::uuid,$2,$3,$4,$5,$6)
		 RETURNING id::text, event_id::text, name, position, color, schedule, unlock_rule, created_at`,
		eventID, name, position, color, nilJSON(schedule), nilJSON(unlock)).
		Scan(&b.ID, &b.EventID, &b.Name, &b.Position, &b.Color, &sc, &ur, &b.CreatedAt)
	if isUnique(err) {
		return Block{}, ErrConflict
	}
	b.Schedule, b.UnlockRule = sc, ur
	return b, err
}

func (s *Store) ListBlocks(ctx context.Context, eventID string) ([]Block, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id::text, event_id::text, name, position, color, schedule, unlock_rule, created_at
		 FROM blocks WHERE event_id=$1::uuid ORDER BY position, name`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Block
	for rows.Next() {
		var b Block
		var sc, ur []byte
		if err := rows.Scan(&b.ID, &b.EventID, &b.Name, &b.Position, &b.Color, &sc, &ur, &b.CreatedAt); err != nil {
			return nil, err
		}
		b.Schedule, b.UnlockRule = sc, ur
		out = append(out, b)
	}
	return out, rows.Err()
}

// ---------------- bulk actions ----------------

func (s *Store) ResolveSelector(ctx context.Context, eventID, mode string, ids []string, filter map[string]any, exclude []string) ([]string, error) {
	var candidate []string
	if mode == "ids" {
		candidate = ids
	} else {
		q := `SELECT id::text FROM event_challenges WHERE event_id=$1::uuid`
		args := []any{eventID}
		if v := getStr(filter, "state"); v != "" {
			args = append(args, v)
			q += fmt.Sprintf(` AND state=$%d`, len(args))
		}
		if v := getStr(filter, "category"); v != "" {
			args = append(args, v)
			q += fmt.Sprintf(` AND category=$%d`, len(args))
		}
		if v, ok := filter["block_id"]; ok {
			if sv, _ := v.(string); sv == "none" || sv == "" {
				q += ` AND block_id IS NULL`
			} else {
				args = append(args, sv)
				q += fmt.Sprintf(` AND block_id=$%d::uuid`, len(args))
			}
		}
		if tags := getStrSlice(filter, "tags"); len(tags) > 0 {
			args = append(args, tags)
			q += fmt.Sprintf(` AND tags @> $%d`, len(args))
		}
		rows, err := s.pool.Query(ctx, q, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return nil, err
			}
			candidate = append(candidate, id)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	ex := map[string]bool{}
	for _, e := range exclude {
		ex[e] = true
	}
	out := make([]string, 0, len(candidate))
	for _, id := range candidate {
		if !ex[id] {
			out = append(out, id)
		}
	}
	return out, nil
}

// BulkApply applies an action to the given event_challenge ids, snapshots undo
// state for reversible actions, and records a bulk_job + audit entry.
func (s *Store) BulkApply(ctx context.Context, eventID, actorID, action string, params map[string]any, ecIDs []string) (string, int, error) {
	if len(ecIDs) == 0 {
		return "", 0, nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", 0, err
	}
	defer tx.Rollback(ctx)

	// snapshot for undo (state, block_id, tags) unless delete
	var undo []byte
	if action != "delete" {
		snap, err := snapshot(ctx, tx, ecIDs)
		if err != nil {
			return "", 0, err
		}
		undo = snap
	}

	var affected int
	switch action {
	case "publish", "hide", "archive", "set_state":
		state := map[string]string{"publish": "published", "hide": "hidden", "archive": "archived"}[action]
		if action == "set_state" {
			state = getStr(params, "state")
		}
		ct, err := tx.Exec(ctx, `UPDATE event_challenges SET state=$1 WHERE id = ANY($2::uuid[])`, state, ecIDs)
		if err != nil {
			return "", 0, err
		}
		affected = int(ct.RowsAffected())
	case "assign_block":
		blockID := getStr(params, "block_id")
		ct, err := tx.Exec(ctx, `UPDATE event_challenges SET block_id=$1 WHERE id = ANY($2::uuid[])`, nilIfEmpty(blockID), ecIDs)
		if err != nil {
			return "", 0, err
		}
		affected = int(ct.RowsAffected())
	case "add_tags":
		tags := getStrSlice(params, "tags")
		ct, err := tx.Exec(ctx,
			`UPDATE event_challenges SET tags = array(SELECT DISTINCT e FROM unnest(tags || $1::text[]) e) WHERE id = ANY($2::uuid[])`,
			tags, ecIDs)
		if err != nil {
			return "", 0, err
		}
		affected = int(ct.RowsAffected())
	case "remove_tags":
		tags := getStrSlice(params, "tags")
		ct, err := tx.Exec(ctx,
			`UPDATE event_challenges SET tags = array(SELECT e FROM unnest(tags) e WHERE e <> ALL($1::text[])) WHERE id = ANY($2::uuid[])`,
			tags, ecIDs)
		if err != nil {
			return "", 0, err
		}
		affected = int(ct.RowsAffected())
	case "set_schedule":
		sched := getRawJSON(params, "schedule")
		ct, err := tx.Exec(ctx, `UPDATE event_challenges SET schedule=$1 WHERE id = ANY($2::uuid[])`, nilJSON(sched), ecIDs)
		if err != nil {
			return "", 0, err
		}
		affected = int(ct.RowsAffected())
	case "delete":
		ct, err := tx.Exec(ctx, `DELETE FROM event_challenges WHERE id = ANY($1::uuid[])`, ecIDs)
		if err != nil {
			return "", 0, err
		}
		affected = int(ct.RowsAffected())
	default:
		return "", 0, fmt.Errorf("unknown bulk action %q", action)
	}

	var jobID string
	if err = tx.QueryRow(ctx,
		`INSERT INTO bulk_jobs (event_id, actor_id, action, affected, undo_payload)
		 VALUES ($1::uuid,$2::uuid,$3,$4,$5) RETURNING id::text`,
		eventID, nilIfEmpty(actorID), action, affected, nilJSON(undo)).Scan(&jobID); err != nil {
		return "", 0, err
	}
	meta, _ := json.Marshal(map[string]any{"action": action, "affected": affected, "count": len(ecIDs)})
	if _, err = tx.Exec(ctx,
		`INSERT INTO audit_log (event_id, actor_id, action, target, meta) VALUES ($1::uuid,$2::uuid,$3,$4,$5::jsonb)`,
		eventID, nilIfEmpty(actorID), "bulk:"+action, jobID, string(meta)); err != nil {
		return "", 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return "", 0, err
	}
	return jobID, affected, nil
}

type snapRow struct {
	ID      string   `json:"id"`
	State   string   `json:"state"`
	BlockID *string  `json:"block_id"`
	Tags    []string `json:"tags"`
}

func snapshot(ctx context.Context, tx pgx.Tx, ecIDs []string) ([]byte, error) {
	rows, err := tx.Query(ctx, `SELECT id::text, state, block_id::text, tags FROM event_challenges WHERE id = ANY($1::uuid[])`, ecIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var snap []snapRow
	for rows.Next() {
		var r snapRow
		if err := rows.Scan(&r.ID, &r.State, &r.BlockID, &r.Tags); err != nil {
			return nil, err
		}
		snap = append(snap, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return json.Marshal(snap)
}

// BulkUndo restores state/block/tags captured in a bulk job's snapshot.
func (s *Store) BulkUndo(ctx context.Context, jobID string) (int, error) {
	var raw []byte
	var undone bool
	err := s.pool.QueryRow(ctx, `SELECT undo_payload, undone FROM bulk_jobs WHERE id=$1::uuid`, jobID).Scan(&raw, &undone)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	if undone || len(raw) == 0 {
		return 0, ErrConflict
	}
	var snap []snapRow
	if err := json.Unmarshal(raw, &snap); err != nil {
		return 0, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	n := 0
	for _, r := range snap {
		if _, err := tx.Exec(ctx, `UPDATE event_challenges SET state=$2, block_id=$3, tags=$4 WHERE id=$1::uuid`,
			r.ID, r.State, nilPtr(r.BlockID), r.Tags); err != nil {
			return 0, err
		}
		n++
	}
	if _, err := tx.Exec(ctx, `UPDATE bulk_jobs SET undone=true WHERE id=$1::uuid`, jobID); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return n, nil
}

// ---------------- saved views + audit ----------------

func (s *Store) CreateSavedView(ctx context.Context, eventID, userID, name string, filter json.RawMessage) (string, error) {
	f := "{}"
	if len(filter) > 0 {
		f = string(filter)
	}
	var id string
	err := s.pool.QueryRow(ctx,
		`INSERT INTO saved_views (event_id, user_id, name, filter) VALUES ($1::uuid,$2::uuid,$3,$4::jsonb)
		 ON CONFLICT (event_id, user_id, name) DO UPDATE SET filter=EXCLUDED.filter RETURNING id::text`,
		eventID, userID, name, f).Scan(&id)
	return id, err
}

func (s *Store) ListSavedViews(ctx context.Context, eventID, userID string) ([]map[string]any, error) {
	rows, err := s.pool.Query(ctx, `SELECT id::text, name, filter FROM saved_views WHERE event_id=$1::uuid AND user_id=$2::uuid ORDER BY name`, eventID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, name string
		var filter []byte
		if err := rows.Scan(&id, &name, &filter); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "name": name, "filter": json.RawMessage(filter)})
	}
	return out, rows.Err()
}

// ---------------- helpers ----------------

func nilJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return string(b)
}

func nilPtr(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func getStr(m map[string]any, k string) string {
	if v, ok := m[k]; ok {
		if sv, ok := v.(string); ok {
			return sv
		}
	}
	return ""
}

func getStrSlice(m map[string]any, k string) []string {
	v, ok := m[k]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		if sv, ok := e.(string); ok {
			out = append(out, sv)
		}
	}
	return out
}

func getRawJSON(m map[string]any, k string) []byte {
	v, ok := m[k]
	if !ok {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}
