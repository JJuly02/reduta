package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (s *Store) CreateTeam(ctx context.Context, orgID, name, inviteCode string) (Team, error) {
	var t Team
	err := s.pool.QueryRow(ctx,
		`INSERT INTO teams (org_id, name, invite_code) VALUES ($1::uuid,$2,$3)
		 RETURNING id::text, name, invite_code, created_at`,
		orgID, name, inviteCode).Scan(&t.ID, &t.Name, &t.InviteCode, &t.CreatedAt)
	if isUnique(err) {
		return Team{}, ErrConflict
	}
	return t, err
}

func (s *Store) TeamByInvite(ctx context.Context, code string) (Team, error) {
	var t Team
	err := s.pool.QueryRow(ctx,
		`SELECT id::text, name, invite_code, created_at FROM teams WHERE invite_code=$1`, code).
		Scan(&t.ID, &t.Name, &t.InviteCode, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Team{}, ErrNotFound
	}
	return t, err
}

// AddTeamMember adds a user to a team; ErrConflict if the user already has one.
func (s *Store) AddTeamMember(ctx context.Context, userID, teamID, role string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO team_members (user_id, team_id, role) VALUES ($1::uuid,$2::uuid,$3)`,
		userID, teamID, role)
	if isUnique(err) {
		return ErrConflict
	}
	return err
}

func (s *Store) UserTeam(ctx context.Context, userID string) (Team, error) {
	var t Team
	err := s.pool.QueryRow(ctx,
		`SELECT t.id::text, t.name, t.invite_code, tm.role, t.created_at
		 FROM team_members tm JOIN teams t ON t.id = tm.team_id WHERE tm.user_id=$1::uuid`, userID).
		Scan(&t.ID, &t.Name, &t.InviteCode, &t.Role, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Team{}, ErrNotFound
	}
	return t, err
}

func (s *Store) ListOrgTeams(ctx context.Context, orgID string) ([]Team, error) {
	rows, err := s.pool.Query(ctx, `SELECT id::text, name, created_at FROM teams WHERE org_id=$1::uuid ORDER BY name`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Team{}
	for rows.Next() {
		var t Team
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListTeams returns the teams assigned to an event (scoreboard, plugin, admin).
func (s *Store) ListTeams(ctx context.Context, eventID string) ([]Team, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT t.id::text, t.name, t.created_at FROM event_teams et JOIN teams t ON t.id=et.team_id
		 WHERE et.event_id=$1::uuid ORDER BY t.name`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Team{}
	for rows.Next() {
		var t Team
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) AssignEventTeam(ctx context.Context, eventID, teamID string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO event_teams (event_id, team_id) VALUES ($1::uuid,$2::uuid) ON CONFLICT DO NOTHING`,
		eventID, teamID)
	return err
}

func (s *Store) UnassignEventTeam(ctx context.Context, eventID, teamID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM event_teams WHERE event_id=$1::uuid AND team_id=$2::uuid`, eventID, teamID)
	return err
}

// PlayerTeamForEvent returns the caller's team id if their global team is
// assigned to the event, else ErrNotFound.
func (s *Store) PlayerTeamForEvent(ctx context.Context, eventID, userID string) (string, error) {
	var teamID string
	err := s.pool.QueryRow(ctx,
		`SELECT tm.team_id::text FROM team_members tm
		 JOIN event_teams et ON et.team_id = tm.team_id AND et.event_id=$1::uuid
		 WHERE tm.user_id=$2::uuid`, eventID, userID).Scan(&teamID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return teamID, err
}

// ListEventsForTeam returns events the team is assigned to (player view).
func (s *Store) ListEventsForTeam(ctx context.Context, orgID, teamID string) ([]Event, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT e.id::text, e.org_id::text, e.slug, e.name, e.state, e.starts_at, e.ends_at, e.freeze_at, e.first_blood_bonus, e.created_at
		 FROM events e JOIN event_teams et ON et.event_id=e.id
		 WHERE e.org_id=$1::uuid AND et.team_id=$2::uuid ORDER BY e.created_at DESC`, orgID, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.OrgID, &e.Slug, &e.Name, &e.State, &e.StartsAt, &e.EndsAt, &e.FreezeAt, &e.FirstBloodBonus, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
