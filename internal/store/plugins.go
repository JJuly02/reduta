package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

type Plugin struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	BaseURL      string          `json:"base_url"`
	Secret       string          `json:"-"`
	Capabilities json.RawMessage `json:"capabilities"`
	Events       json.RawMessage `json:"events"`
}

func (s *Store) RegisterPlugin(ctx context.Context, id, name, baseURL, secret string, tokenHash []byte, capabilities, events json.RawMessage) error {
	if len(capabilities) == 0 {
		capabilities = json.RawMessage(`[]`)
	}
	if len(events) == 0 {
		events = json.RawMessage(`[]`)
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO plugins (id, name, base_url, secret, token_hash, capabilities, events)
		 VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb)
		 ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, base_url=EXCLUDED.base_url,
		   secret=EXCLUDED.secret, token_hash=EXCLUDED.token_hash,
		   capabilities=EXCLUDED.capabilities, events=EXCLUDED.events`,
		id, name, baseURL, secret, tokenHash, string(capabilities), string(events))
	return err
}

func (s *Store) PluginByToken(ctx context.Context, tokenHash []byte) (Plugin, error) {
	var p Plugin
	var caps, evs []byte
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, base_url, secret, capabilities, events FROM plugins WHERE token_hash=$1`, tokenHash).
		Scan(&p.ID, &p.Name, &p.BaseURL, &p.Secret, &caps, &evs)
	if errors.Is(err, pgx.ErrNoRows) {
		return Plugin{}, ErrNotFound
	}
	p.Capabilities, p.Events = caps, evs
	return p, err
}

func (s *Store) ListPlugins(ctx context.Context) ([]Plugin, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, base_url, secret, capabilities, events FROM plugins ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Plugin
	for rows.Next() {
		var p Plugin
		var caps, evs []byte
		if err := rows.Scan(&p.ID, &p.Name, &p.BaseURL, &p.Secret, &caps, &evs); err != nil {
			return nil, err
		}
		p.Capabilities, p.Events = caps, evs
		out = append(out, p)
	}
	return out, rows.Err()
}

// PluginAward records an idempotent score event from a plugin and adjusts the
// scoreboard. Returns false (no-op) if the (event, source, ref_id) already exists.
func (s *Store) PluginAward(ctx context.Context, eventID, teamID, source, refID string, points int, meta json.RawMessage) (bool, error) {
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ct, err := tx.Exec(ctx,
		`INSERT INTO score_events (event_id, team_id, kind, source, ref_id, points, meta)
		 VALUES ($1::uuid,$2::uuid,'award',$3,$4,$5,$6::jsonb)
		 ON CONFLICT (event_id, source, ref_id) DO NOTHING`,
		eventID, teamID, source, refID, points, string(meta))
	if err != nil {
		return false, err
	}
	if ct.RowsAffected() == 0 {
		return false, tx.Commit(ctx) // idempotent no-op
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO scoreboard_entries (event_id, team_id, points, solves)
		 VALUES ($1::uuid,$2::uuid,$3,0)
		 ON CONFLICT (event_id, team_id) DO UPDATE SET points = scoreboard_entries.points + EXCLUDED.points`,
		eventID, teamID, points); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// DeletePluginAward reverses a previously recorded award (spec 7.3).
func (s *Store) DeletePluginAward(ctx context.Context, eventID, source, refID string) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var teamID string
	var points int
	err = tx.QueryRow(ctx,
		`DELETE FROM score_events WHERE event_id=$1::uuid AND source=$2 AND ref_id=$3
		 RETURNING team_id::text, points`, eventID, source, refID).Scan(&teamID, &points)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, tx.Commit(ctx)
	}
	if err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE scoreboard_entries SET points = points - $3 WHERE event_id=$1::uuid AND team_id=$2::uuid`,
		eventID, teamID, points); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}
