package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// ChallengeFileMeta describes an attached file without its bytes (list/detail).
type ChallengeFileMeta struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	SHA256      []byte `json:"-"`
}

// ChallengeFileInput is one file to store (bytes plus precomputed digest/size).
type ChallengeFileInput struct {
	Name        string
	ContentType string
	Data        []byte
	SHA256      []byte
	Size        int64
}

// ChallengeFileContent is a stored file with its bytes, for download.
type ChallengeFileContent struct {
	ECID        string
	Name        string
	ContentType string
	Size        int64
	Data        []byte
}

// ListChallengeFiles returns the file metadata for a challenge, ordered by name.
func (s *Store) ListChallengeFiles(ctx context.Context, ecID string) ([]ChallengeFileMeta, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id::text, name, content_type, size, sha256
		 FROM event_challenge_files WHERE ec_id=$1::uuid ORDER BY name`, ecID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChallengeFileMeta{}
	for rows.Next() {
		var f ChallengeFileMeta
		if err := rows.Scan(&f.ID, &f.Name, &f.ContentType, &f.Size, &f.SHA256); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// GetChallengeFile returns one file with its bytes (for the download endpoint).
func (s *Store) GetChallengeFile(ctx context.Context, fileID string) (ChallengeFileContent, error) {
	var f ChallengeFileContent
	err := s.pool.QueryRow(ctx,
		`SELECT ec_id::text, name, content_type, size, data
		 FROM event_challenge_files WHERE id=$1::uuid`, fileID).
		Scan(&f.ECID, &f.Name, &f.ContentType, &f.Size, &f.Data)
	if errors.Is(err, pgx.ErrNoRows) {
		return ChallengeFileContent{}, ErrNotFound
	}
	return f, err
}

// ReplaceChallengeFiles swaps the entire file set for a challenge in one tx.
func (s *Store) ReplaceChallengeFiles(ctx context.Context, ecID string, files []ChallengeFileInput) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM event_challenge_files WHERE ec_id=$1::uuid`, ecID); err != nil {
		return err
	}
	for _, f := range files {
		if _, err := tx.Exec(ctx,
			`INSERT INTO event_challenge_files (ec_id, name, content_type, size, sha256, data)
			 VALUES ($1::uuid,$2,$3,$4,$5,$6)`,
			ecID, f.Name, f.ContentType, f.Size, f.SHA256, f.Data); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
