package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

// FindECByTitle finds an event challenge by (event, title) for idempotent import.
func (s *Store) FindECByTitle(ctx context.Context, eventID, title string) (EventChallenge, error) {
	var c EventChallenge
	var raw, sc, ur []byte
	err := s.pool.QueryRow(ctx,
		`SELECT id::text, event_id::text, title, category, description_md, state, scoring, block_id::text, tags, schedule, unlock_rule, position, created_at
		 FROM event_challenges WHERE event_id=$1::uuid AND title=$2 LIMIT 1`, eventID, title).
		Scan(&c.ID, &c.EventID, &c.Title, &c.Category, &c.DescriptionMD, &c.State, &raw, &c.BlockID, &c.Tags, &sc, &ur, &c.Position, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return EventChallenge{}, ErrNotFound
	}
	c.Scoring, c.Schedule, c.UnlockRule = raw, sc, ur
	return c, err
}

// UpdateChallengeContent updates the mutable content fields of an event challenge.
func (s *Store) UpdateChallengeContent(ctx context.Context, ecID, category, descMD string, scoring json.RawMessage, state string, tags []string) error {
	if len(scoring) == 0 {
		scoring = json.RawMessage(`{"type":"static","points":100}`)
	}
	if tags == nil {
		tags = []string{}
	}
	ct, err := s.pool.Exec(ctx,
		`UPDATE event_challenges SET category=$2, description_md=$3, scoring=$4::jsonb, state=$5, tags=$6 WHERE id=$1::uuid`,
		ecID, category, descMD, string(scoring), state, tags)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ReplaceFlags atomically swaps a challenge's flag set.
func (s *Store) ReplaceFlags(ctx context.Context, ecID string, flags []Flag) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM flags WHERE ec_id=$1::uuid`, ecID); err != nil {
		return err
	}
	for _, f := range flags {
		if _, err := tx.Exec(ctx, `INSERT INTO flags (ec_id, value_hash, case_sensitive) VALUES ($1::uuid,$2,$3)`, ecID, f.ValueHash, f.CaseSensitive); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// CreateEmbeddedChallenge creates a standalone event challenge (used by import),
// optionally attaching flags in one transaction.
func (s *Store) CreateChallengeWithFlags(ctx context.Context, eventID, title, category, descMD string, scoring json.RawMessage, state string, tags []string, flags []Flag) (string, error) {
	if len(scoring) == 0 {
		scoring = json.RawMessage(`{"type":"static","points":100}`)
	}
	if tags == nil {
		tags = []string{}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var id string
	if err := tx.QueryRow(ctx,
		`INSERT INTO event_challenges (event_id, title, category, description_md, scoring, state, tags)
		 VALUES ($1::uuid,$2,$3,$4,$5::jsonb,$6,$7) RETURNING id::text`,
		eventID, title, category, descMD, string(scoring), state, tags).Scan(&id); err != nil {
		return "", err
	}
	for _, f := range flags {
		if _, err := tx.Exec(ctx, `INSERT INTO flags (ec_id, value_hash, case_sensitive) VALUES ($1::uuid,$2,$3)`, id, f.ValueHash, f.CaseSensitive); err != nil {
			return "", err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return id, nil
}
