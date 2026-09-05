package store

import (
	"context"
	"time"
)

type SeriesPoint struct {
	T     time.Time `json:"t"`
	Score int       `json:"score"`
}

type TeamSeries struct {
	TeamID string        `json:"team_id"`
	Name   string        `json:"name"`
	Points int           `json:"points"`
	Data   []SeriesPoint `json:"data"`
}

// ScoreboardSeries returns cumulative score over time for the top N teams,
// built from score_events (spec 5.7 progress chart).
func (s *Store) ScoreboardSeries(ctx context.Context, eventID string, topN int) ([]TeamSeries, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT t.id::text, t.name, se.points
		 FROM scoreboard_entries se JOIN teams t ON t.id = se.team_id
		 WHERE se.event_id=$1::uuid AND se.points > 0
		 ORDER BY se.points DESC, se.last_solve_at ASC NULLS LAST
		 LIMIT $2`, eventID, topN)
	if err != nil {
		return nil, err
	}
	var teams []TeamSeries
	idx := map[string]int{}
	for rows.Next() {
		var ts TeamSeries
		if err := rows.Scan(&ts.TeamID, &ts.Name, &ts.Points); err != nil {
			rows.Close()
			return nil, err
		}
		idx[ts.TeamID] = len(teams)
		teams = append(teams, ts)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(teams) == 0 {
		return []TeamSeries{}, nil
	}

	ids := make([]string, len(teams))
	for i, t := range teams {
		ids[i] = t.TeamID
	}
	ev, err := s.pool.Query(ctx,
		`SELECT team_id::text, occurred_at, points FROM score_events
		 WHERE event_id=$1::uuid AND team_id = ANY($2::uuid[])
		 ORDER BY occurred_at ASC`, eventID, ids)
	if err != nil {
		return nil, err
	}
	defer ev.Close()

	type row struct {
		team string
		at   time.Time
		pts  int
	}
	var all []row
	var t0 time.Time
	for ev.Next() {
		var r row
		if err := ev.Scan(&r.team, &r.at, &r.pts); err != nil {
			return nil, err
		}
		if t0.IsZero() || r.at.Before(t0) {
			t0 = r.at
		}
		all = append(all, r)
	}
	if err := ev.Err(); err != nil {
		return nil, err
	}
	// start every line at zero at t0 so the chart begins at the origin
	for i := range teams {
		teams[i].Data = []SeriesPoint{{T: t0, Score: 0}}
	}
	running := map[string]int{}
	for _, r := range all {
		running[r.team] += r.pts
		if i, ok := idx[r.team]; ok {
			teams[i].Data = append(teams[i].Data, SeriesPoint{T: r.at, Score: running[r.team]})
		}
	}
	return teams, nil
}
