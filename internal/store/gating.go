package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (s *Store) SetChallengeSchedule(ctx context.Context, ecID string, schedule json.RawMessage) error {
	ct, err := s.pool.Exec(ctx, `UPDATE event_challenges SET schedule=$2 WHERE id=$1::uuid`, ecID, nilJSON(schedule))
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SetChallengeUnlock(ctx context.Context, ecID string, rule json.RawMessage) error {
	ct, err := s.pool.Exec(ctx, `UPDATE event_challenges SET unlock_rule=$2 WHERE id=$1::uuid`, ecID, nilJSON(rule))
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SetBlockSchedule(ctx context.Context, blockID string, schedule json.RawMessage) error {
	ct, err := s.pool.Exec(ctx, `UPDATE blocks SET schedule=$2 WHERE id=$1::uuid`, blockID, nilJSON(schedule))
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) GetBlock(ctx context.Context, id string) (Block, error) {
	var b Block
	var sc, ur []byte
	err := s.pool.QueryRow(ctx,
		`SELECT id::text, event_id::text, name, position, color, schedule, unlock_rule, created_at
		 FROM blocks WHERE id=$1::uuid`, id).
		Scan(&b.ID, &b.EventID, &b.Name, &b.Position, &b.Color, &sc, &ur, &b.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Block{}, ErrNotFound
	}
	b.Schedule, b.UnlockRule = sc, ur
	return b, err
}

// TeamGateState holds everything the unlock evaluator needs for one team.
type TeamGateState struct {
	SolvedEC    []string
	Points      int
	BlockSolved map[string]int
	BlockTotal  map[string]int
}

func (s *Store) TeamGateState(ctx context.Context, eventID, teamID string) (TeamGateState, error) {
	st := TeamGateState{BlockSolved: map[string]int{}, BlockTotal: map[string]int{}}

	rows, err := s.pool.Query(ctx,
		`SELECT s.ec_id::text, ec.block_id::text
		 FROM submissions s JOIN event_challenges ec ON ec.id = s.ec_id
		 WHERE s.event_id=$1::uuid AND s.team_id=$2::uuid AND s.correct`, eventID, teamID)
	if err != nil {
		return st, err
	}
	for rows.Next() {
		var ecID string
		var block *string
		if err := rows.Scan(&ecID, &block); err != nil {
			rows.Close()
			return st, err
		}
		st.SolvedEC = append(st.SolvedEC, ecID)
		if block != nil {
			st.BlockSolved[*block]++
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return st, err
	}

	if err := s.pool.QueryRow(ctx,
		`SELECT points FROM scoreboard_entries WHERE event_id=$1::uuid AND team_id=$2::uuid`, eventID, teamID).
		Scan(&st.Points); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return st, err
	}

	rows2, err := s.pool.Query(ctx,
		`SELECT block_id::text, count(*) FROM event_challenges
		 WHERE event_id=$1::uuid AND state='published' AND block_id IS NOT NULL GROUP BY block_id`, eventID)
	if err != nil {
		return st, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var block string
		var n int
		if err := rows2.Scan(&block, &n); err != nil {
			return st, err
		}
		st.BlockTotal[block] = n
	}
	return st, rows2.Err()
}
