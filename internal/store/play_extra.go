package store

import (
	"context"
	"time"
)

// SolveCounts returns solves per event_challenge for an event (ec_id -> count).
// A solve is a correct submission (unique per ec_id, team_id via solves_unique).
func (s *Store) SolveCounts(ctx context.Context, eventID string) (map[string]int, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT ec_id::text, count(*) FROM submissions
		 WHERE event_id=$1::uuid AND correct GROUP BY ec_id`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]int{}
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		m[id] = n
	}
	return m, rows.Err()
}

// AttemptRow is one of a team's submissions against a challenge. Raw wrong flags
// are never stored, so only the verdict and timestamp are exposed.
type AttemptRow struct {
	Correct     bool      `json:"correct"`
	SubmittedAt time.Time `json:"submitted_at"`
}

// TeamChallengeAttempts lists a team's submissions for one challenge, newest first.
func (s *Store) TeamChallengeAttempts(ctx context.Context, ecID, teamID string) ([]AttemptRow, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT correct, submitted_at FROM submissions
		 WHERE ec_id=$1::uuid AND team_id=$2::uuid
		 ORDER BY submitted_at DESC LIMIT 50`, ecID, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AttemptRow{}
	for rows.Next() {
		var a AttemptRow
		if err := rows.Scan(&a.Correct, &a.SubmittedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
