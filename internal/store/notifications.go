package store

import (
	"context"
	"time"
)

type Notification struct {
	ID        string    `json:"id"`
	EventID   string    `json:"event_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Store) CreateNotification(ctx context.Context, eventID, title, content string) (Notification, error) {
	var n Notification
	err := s.pool.QueryRow(ctx,
		`INSERT INTO notifications (event_id, title, content) VALUES ($1::uuid,$2,$3)
		 RETURNING id::text, event_id::text, title, content, created_at`,
		eventID, title, content).Scan(&n.ID, &n.EventID, &n.Title, &n.Content, &n.CreatedAt)
	return n, err
}

func (s *Store) ListNotifications(ctx context.Context, eventID string, limit int) ([]Notification, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id::text, event_id::text, title, content, created_at FROM notifications
		 WHERE event_id=$1::uuid ORDER BY created_at DESC LIMIT $2`, eventID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.EventID, &n.Title, &n.Content, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

type SubmissionRow struct {
	ID          int64     `json:"id"`
	UserName    string    `json:"user"`
	TeamName    string    `json:"team"`
	Challenge   string    `json:"challenge"`
	Correct     bool      `json:"correct"`
	SubmittedAt time.Time `json:"submitted_at"`
}

func (s *Store) ListSubmissions(ctx context.Context, eventID string, limit int) ([]SubmissionRow, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT s.id, u.display_name, t.name, ec.title, s.correct, s.submitted_at
		 FROM submissions s
		 JOIN users u ON u.id = s.user_id
		 JOIN teams t ON t.id = s.team_id
		 JOIN event_challenges ec ON ec.id = s.ec_id
		 WHERE s.event_id=$1::uuid ORDER BY s.submitted_at DESC LIMIT $2`, eventID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SubmissionRow{}
	for rows.Next() {
		var r SubmissionRow
		if err := rows.Scan(&r.ID, &r.UserName, &r.TeamName, &r.Challenge, &r.Correct, &r.SubmittedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type EventStats struct {
	Teams       int `json:"teams"`
	Challenges  int `json:"challenges"`
	Published   int `json:"published"`
	Solves      int `json:"solves"`
	Submissions int `json:"submissions"`
}

func (s *Store) Stats(ctx context.Context, eventID string) (EventStats, error) {
	var st EventStats
	err := s.pool.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM event_teams WHERE event_id=$1::uuid),
		  (SELECT count(*) FROM event_challenges WHERE event_id=$1::uuid),
		  (SELECT count(*) FROM event_challenges WHERE event_id=$1::uuid AND state='published'),
		  (SELECT count(*) FROM submissions WHERE event_id=$1::uuid AND correct),
		  (SELECT count(*) FROM submissions WHERE event_id=$1::uuid)
	`, eventID).Scan(&st.Teams, &st.Challenges, &st.Published, &st.Solves, &st.Submissions)
	return st, err
}
