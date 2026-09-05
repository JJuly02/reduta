package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type Instance struct {
	ID        string    `json:"id"`
	ECID      string    `json:"ec_id"`
	TeamID    string    `json:"team_id"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	Status    string    `json:"status"`
	Extends   int       `json:"extends"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Store) SetInstanceSpec(ctx context.Context, ecID string, spec json.RawMessage) error {
	ct, err := s.pool.Exec(ctx, `UPDATE event_challenges SET instance_spec=$2 WHERE id=$1::uuid`, ecID, nilJSON(spec))
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ChallengeInstanceSpec(ctx context.Context, ecID string) (json.RawMessage, error) {
	var spec []byte
	err := s.pool.QueryRow(ctx, `SELECT instance_spec FROM event_challenges WHERE id=$1::uuid`, ecID).Scan(&spec)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return spec, err
}

func (s *Store) GetActiveInstance(ctx context.Context, ecID, teamID string) (Instance, error) {
	var i Instance
	err := s.pool.QueryRow(ctx,
		`SELECT id::text, ec_id::text, team_id::text, host, port, status, extends, expires_at, created_at
		 FROM instances WHERE ec_id=$1::uuid AND team_id=$2::uuid AND status='running'`, ecID, teamID).
		Scan(&i.ID, &i.ECID, &i.TeamID, &i.Host, &i.Port, &i.Status, &i.Extends, &i.ExpiresAt, &i.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Instance{}, ErrNotFound
	}
	return i, err
}

func (s *Store) CountRunningInstances(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM instances WHERE status='running'`).Scan(&n)
	return n, err
}

func (s *Store) CreateInstance(ctx context.Context, ecID, teamID, host string, port int, expiresAt time.Time) (Instance, error) {
	var i Instance
	err := s.pool.QueryRow(ctx,
		`INSERT INTO instances (ec_id, team_id, host, port, expires_at) VALUES ($1::uuid,$2::uuid,$3,$4,$5)
		 RETURNING id::text, ec_id::text, team_id::text, host, port, status, extends, expires_at, created_at`,
		ecID, teamID, host, port, expiresAt).
		Scan(&i.ID, &i.ECID, &i.TeamID, &i.Host, &i.Port, &i.Status, &i.Extends, &i.ExpiresAt, &i.CreatedAt)
	if isUnique(err) {
		return Instance{}, ErrConflict
	}
	return i, err
}

func (s *Store) ExtendInstance(ctx context.Context, ecID, teamID string, extra time.Duration, maxExtends int) (Instance, error) {
	var i Instance
	err := s.pool.QueryRow(ctx,
		`UPDATE instances SET expires_at = expires_at + $3, extends = extends + 1
		 WHERE ec_id=$1::uuid AND team_id=$2::uuid AND status='running' AND extends < $4
		 RETURNING id::text, ec_id::text, team_id::text, host, port, status, extends, expires_at, created_at`,
		ecID, teamID, extra, maxExtends).
		Scan(&i.ID, &i.ECID, &i.TeamID, &i.Host, &i.Port, &i.Status, &i.Extends, &i.ExpiresAt, &i.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Instance{}, ErrNotFound
	}
	return i, err
}

func (s *Store) DestroyInstance(ctx context.Context, ecID, teamID string) (bool, error) {
	ct, err := s.pool.Exec(ctx,
		`UPDATE instances SET status='destroyed' WHERE ec_id=$1::uuid AND team_id=$2::uuid AND status='running'`, ecID, teamID)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}
