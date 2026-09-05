// Package store is the data-access layer over pgx. Queries are hand-written to
// keep the SQL explicit and free of N+1 (see docs/adr/0007-sqlc-deferred.md for
// why sqlc is introduced later). UUIDs cross the boundary as strings, cast at
// the SQL edge (::uuid), to avoid type-mapping ceremony in M1.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ---------------- models ----------------

type User struct {
	ID           string    `json:"id"`
	OrgID        string    `json:"org_id"`
	Email        string    `json:"email"`
	DisplayName  string    `json:"display_name"`
	Role         string    `json:"role"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Event struct {
	ID              string     `json:"id"`
	OrgID           string     `json:"org_id"`
	Slug            string     `json:"slug"`
	Name            string     `json:"name"`
	State           string     `json:"state"`
	StartsAt        *time.Time `json:"starts_at"`
	EndsAt          *time.Time `json:"ends_at"`
	FreezeAt        *time.Time `json:"freeze_at"`
	FirstBloodBonus int        `json:"first_blood_bonus"`
	CreatedAt       time.Time  `json:"created_at"`
}

type Team struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	InviteCode string    `json:"invite_code,omitempty"`
	Role       string    `json:"role,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type EventChallenge struct {
	ID            string          `json:"id"`
	EventID       string          `json:"event_id"`
	Title         string          `json:"title"`
	Category      string          `json:"category"`
	DescriptionMD string          `json:"description_md"`
	State         string          `json:"state"`
	Scoring       json.RawMessage `json:"scoring"`
	BlockID       *string         `json:"block_id"`
	Tags          []string        `json:"tags"`
	Schedule      json.RawMessage `json:"schedule"`
	UnlockRule    json.RawMessage `json:"unlock_rule"`
	Position      int             `json:"position"`
	CreatedAt     time.Time       `json:"created_at"`
}

type Flag struct {
	ID            int64
	ECID          string
	ValueHash     []byte
	CaseSensitive bool
}

type ScoreRow struct {
	Rank        int        `json:"rank"`
	TeamID      string     `json:"team_id"`
	Name        string     `json:"name"`
	Points      int        `json:"points"`
	Solves      int        `json:"solves"`
	LastSolveAt *time.Time `json:"last_solve_at"`
}

// ---------------- orgs / users ----------------

func (s *Store) DefaultOrgID(ctx context.Context) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT id::text FROM organizations WHERE slug='default'`).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return id, err
}

func (s *Store) CreateUser(ctx context.Context, orgID, email, displayName, passwordHash, role string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (org_id, email, display_name, password_hash, role)
		 VALUES ($1::uuid,$2,$3,$4,$5)
		 RETURNING id::text, org_id::text, email, display_name, role, created_at`,
		orgID, email, displayName, passwordHash, role).
		Scan(&u.ID, &u.OrgID, &u.Email, &u.DisplayName, &u.Role, &u.CreatedAt)
	if isUnique(err) {
		return User{}, ErrConflict
	}
	return u, err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`SELECT id::text, org_id::text, email, display_name, role, password_hash, created_at
		 FROM users WHERE email=$1`, email).
		Scan(&u.ID, &u.OrgID, &u.Email, &u.DisplayName, &u.Role, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) GetUserByID(ctx context.Context, id string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`SELECT id::text, org_id::text, email, display_name, role, created_at
		 FROM users WHERE id=$1::uuid`, id).
		Scan(&u.ID, &u.OrgID, &u.Email, &u.DisplayName, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

// ---------------- sessions ----------------

func (s *Store) CreateSession(ctx context.Context, tokenHash []byte, userID string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1,$2::uuid,$3)`,
		tokenHash, userID, expiresAt)
	return err
}

func (s *Store) SessionUser(ctx context.Context, tokenHash []byte) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`SELECT u.id::text, u.org_id::text, u.email, u.display_name, u.role, u.created_at
		 FROM sessions s JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash=$1 AND s.expires_at > now()`, tokenHash).
		Scan(&u.ID, &u.OrgID, &u.Email, &u.DisplayName, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash []byte) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, tokenHash)
	return err
}

// ---------------- events ----------------

func (s *Store) CreateEvent(ctx context.Context, orgID, slug, name string, startsAt, endsAt, freezeAt *time.Time, firstBloodBonus int) (Event, error) {
	var e Event
	err := s.pool.QueryRow(ctx,
		`INSERT INTO events (org_id, slug, name, starts_at, ends_at, freeze_at, first_blood_bonus)
		 VALUES ($1::uuid,$2,$3,$4,$5,$6,$7)
		 RETURNING id::text, org_id::text, slug, name, state, starts_at, ends_at, freeze_at, first_blood_bonus, created_at`,
		orgID, slug, name, startsAt, endsAt, freezeAt, firstBloodBonus).
		Scan(&e.ID, &e.OrgID, &e.Slug, &e.Name, &e.State, &e.StartsAt, &e.EndsAt, &e.FreezeAt, &e.FirstBloodBonus, &e.CreatedAt)
	if isUnique(err) {
		return Event{}, ErrConflict
	}
	return e, err
}

func (s *Store) GetEvent(ctx context.Context, id string) (Event, error) {
	var e Event
	err := s.pool.QueryRow(ctx,
		`SELECT id::text, org_id::text, slug, name, state, starts_at, ends_at, freeze_at, first_blood_bonus, created_at
		 FROM events WHERE id=$1::uuid`, id).
		Scan(&e.ID, &e.OrgID, &e.Slug, &e.Name, &e.State, &e.StartsAt, &e.EndsAt, &e.FreezeAt, &e.FirstBloodBonus, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Event{}, ErrNotFound
	}
	return e, err
}

func (s *Store) ListEvents(ctx context.Context, orgID string) ([]Event, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id::text, org_id::text, slug, name, state, starts_at, ends_at, freeze_at, first_blood_bonus, created_at
		 FROM events WHERE org_id=$1::uuid ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.OrgID, &e.Slug, &e.Name, &e.State, &e.StartsAt, &e.EndsAt, &e.FreezeAt, &e.FirstBloodBonus, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) SetEventState(ctx context.Context, id, state string) error {
	ct, err := s.pool.Exec(ctx, `UPDATE events SET state=$2 WHERE id=$1::uuid`, id, state)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------------- challenges / flags ----------------

func (s *Store) CreateChallenge(ctx context.Context, eventID, title, category, descMD string, scoring json.RawMessage, state string) (EventChallenge, error) {
	var c EventChallenge
	var raw, sc, ur []byte
	err := s.pool.QueryRow(ctx,
		`INSERT INTO event_challenges (event_id, title, category, description_md, scoring, state)
		 VALUES ($1::uuid,$2,$3,$4,$5::jsonb,$6)
		 RETURNING id::text, event_id::text, title, category, description_md, state, scoring, block_id::text, tags, schedule, unlock_rule, position, created_at`,
		eventID, title, category, descMD, string(scoring), state).
		Scan(&c.ID, &c.EventID, &c.Title, &c.Category, &c.DescriptionMD, &c.State, &raw, &c.BlockID, &c.Tags, &sc, &ur, &c.Position, &c.CreatedAt)
	c.Scoring, c.Schedule, c.UnlockRule = json.RawMessage(raw), json.RawMessage(sc), json.RawMessage(ur)
	return c, err
}

func (s *Store) AddFlag(ctx context.Context, ecID string, valueHash []byte, caseSensitive bool) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO flags (ec_id, value_hash, case_sensitive) VALUES ($1::uuid,$2,$3)`,
		ecID, valueHash, caseSensitive)
	return err
}

func (s *Store) ListChallenges(ctx context.Context, eventID string, publishedOnly bool) ([]EventChallenge, error) {
	q := `SELECT id::text, event_id::text, title, category, description_md, state, scoring, block_id::text, tags, schedule, unlock_rule, position, created_at
	      FROM event_challenges WHERE event_id=$1::uuid`
	if publishedOnly {
		q += ` AND state='published'`
	}
	q += ` ORDER BY position, created_at`
	rows, err := s.pool.Query(ctx, q, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EventChallenge
	for rows.Next() {
		var c EventChallenge
		var raw, sc, ur []byte
		if err := rows.Scan(&c.ID, &c.EventID, &c.Title, &c.Category, &c.DescriptionMD, &c.State, &raw, &c.BlockID, &c.Tags, &sc, &ur, &c.Position, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.Scoring, c.Schedule, c.UnlockRule = json.RawMessage(raw), json.RawMessage(sc), json.RawMessage(ur)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetChallenge(ctx context.Context, ecID string) (EventChallenge, error) {
	var c EventChallenge
	var raw, sc, ur []byte
	err := s.pool.QueryRow(ctx,
		`SELECT id::text, event_id::text, title, category, description_md, state, scoring, block_id::text, tags, schedule, unlock_rule, position, created_at
		 FROM event_challenges WHERE id=$1::uuid`, ecID).
		Scan(&c.ID, &c.EventID, &c.Title, &c.Category, &c.DescriptionMD, &c.State, &raw, &c.BlockID, &c.Tags, &sc, &ur, &c.Position, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return EventChallenge{}, ErrNotFound
	}
	c.Scoring, c.Schedule, c.UnlockRule = json.RawMessage(raw), json.RawMessage(sc), json.RawMessage(ur)
	return c, err
}

func (s *Store) ChallengeFlags(ctx context.Context, ecID string) ([]Flag, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, ec_id::text, value_hash, case_sensitive FROM flags WHERE ec_id=$1::uuid`, ecID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Flag
	for rows.Next() {
		var f Flag
		if err := rows.Scan(&f.ID, &f.ECID, &f.ValueHash, &f.CaseSensitive); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Store) SetChallengeState(ctx context.Context, ecID, state string) error {
	ct, err := s.pool.Exec(ctx, `UPDATE event_challenges SET state=$2 WHERE id=$1::uuid`, ecID, state)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------------- submissions / scoring ----------------

type SolveResult struct {
	Correct       bool
	AlreadySolved bool
	FirstBlood    bool
	Points        int
}

func (s *Store) RecordWrong(ctx context.Context, eventID, ecID, teamID, userID string, valueHash []byte, ip string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO submissions (event_id, ec_id, team_id, user_id, value_hash, correct, ip)
		 VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,false,$6::inet)`,
		eventID, ecID, teamID, userID, valueHash, nilIfEmpty(ip))
	return err
}

// RecordSolve inserts a winning submission and updates score_events +
// scoreboard_entries in one transaction. The partial unique index solves_unique
// makes a repeat solve a no-op (AlreadySolved).
func (s *Store) RecordSolve(ctx context.Context, e Event, ecID, teamID, userID string, valueHash []byte, ip string, points int) (SolveResult, error) {
	res := SolveResult{Correct: true, Points: points}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO submissions (event_id, ec_id, team_id, user_id, value_hash, correct, ip)
		 VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,true,$6::inet)`,
		e.ID, ecID, teamID, userID, valueHash, nilIfEmpty(ip))
	if err != nil {
		if isUnique(err) {
			res.AlreadySolved = true
			return res, nil
		}
		return res, err
	}

	var n int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM submissions WHERE ec_id=$1::uuid AND correct`, ecID).Scan(&n); err != nil {
		return res, err
	}
	res.FirstBlood = n == 1
	total := points

	if _, err = tx.Exec(ctx,
		`INSERT INTO score_events (event_id, team_id, kind, source, ref_id, points)
		 VALUES ($1::uuid,$2::uuid,'solve','core',$3,$4)
		 ON CONFLICT (event_id, source, ref_id) DO NOTHING`,
		e.ID, teamID, "solve:"+ecID+":"+teamID, points); err != nil {
		return res, err
	}

	if res.FirstBlood && e.FirstBloodBonus > 0 {
		if _, err = tx.Exec(ctx,
			`INSERT INTO score_events (event_id, team_id, kind, source, ref_id, points)
			 VALUES ($1::uuid,$2::uuid,'award','core',$3,$4)
			 ON CONFLICT (event_id, source, ref_id) DO NOTHING`,
			e.ID, teamID, "firstblood:"+ecID, e.FirstBloodBonus); err != nil {
			return res, err
		}
		total += e.FirstBloodBonus
	}

	if _, err = tx.Exec(ctx,
		`INSERT INTO scoreboard_entries (event_id, team_id, points, solves, last_solve_at)
		 VALUES ($1::uuid,$2::uuid,$3,1,now())
		 ON CONFLICT (event_id, team_id) DO UPDATE
		 SET points = scoreboard_entries.points + EXCLUDED.points,
		     solves = scoreboard_entries.solves + 1,
		     last_solve_at = now()`,
		e.ID, teamID, total); err != nil {
		return res, err
	}

	res.Points = total
	if err = tx.Commit(ctx); err != nil {
		return res, err
	}
	return res, nil
}

func (s *Store) Scoreboard(ctx context.Context, eventID string, limit int) ([]ScoreRow, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT t.id::text, t.name, COALESCE(se.points,0), COALESCE(se.solves,0), se.last_solve_at
		 FROM event_teams et
		 JOIN teams t ON t.id = et.team_id
		 LEFT JOIN scoreboard_entries se ON se.team_id=t.id AND se.event_id=et.event_id
		 WHERE et.event_id=$1::uuid
		 ORDER BY COALESCE(se.points,0) DESC, se.last_solve_at ASC NULLS LAST, t.name ASC
		 LIMIT $2`, eventID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ScoreRow
	i := 0
	for rows.Next() {
		var r ScoreRow
		if err := rows.Scan(&r.TeamID, &r.Name, &r.Points, &r.Solves, &r.LastSolveAt); err != nil {
			return nil, err
		}
		i++
		r.Rank = i
		out = append(out, r)
	}
	return out, rows.Err()
}
