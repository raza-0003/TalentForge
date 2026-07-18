package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/faizan/ats/internal/domain"
)

// ResumeRepo persists resume metadata.
type ResumeRepo struct{ pool *pgxpool.Pool }

const resumeCols = `id, candidate_user_id, storage_key, file_name, COALESCE(content_type,''), ` +
	`COALESCE(size_bytes,0), is_primary, parsed_at, parsed_data, created_at`

func scanResume(s scanner) (*domain.Resume, error) {
	var r domain.Resume
	var parsed []byte
	if err := s.Scan(&r.ID, &r.CandidateUserID, &r.StorageKey, &r.FileName,
		&r.ContentType, &r.SizeBytes, &r.IsPrimary, &r.ParsedAt, &parsed, &r.CreatedAt); err != nil {
		return nil, mapErr(err)
	}
	if parsed != nil {
		r.ParsedData = json.RawMessage(parsed)
	}
	return &r, nil
}

// Create inserts a resume row and fills generated fields.
func (r *ResumeRepo) Create(ctx context.Context, res *domain.Resume) error {
	const q = `
		INSERT INTO resumes (candidate_user_id, storage_key, file_name, content_type, size_bytes, is_primary)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`
	return mapErr(r.pool.QueryRow(ctx, q,
		res.CandidateUserID, res.StorageKey, res.FileName,
		nullStr(res.ContentType), res.SizeBytes, res.IsPrimary).
		Scan(&res.ID, &res.CreatedAt))
}

func (r *ResumeRepo) GetByID(ctx context.Context, id int64) (*domain.Resume, error) {
	return scanResume(r.pool.QueryRow(ctx, `SELECT `+resumeCols+` FROM resumes WHERE id = $1`, id))
}

func (r *ResumeRepo) ListByCandidate(ctx context.Context, userID int64) ([]domain.Resume, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+resumeCols+` FROM resumes WHERE candidate_user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []domain.Resume
	for rows.Next() {
		res, err := scanResume(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *res)
	}
	return out, mapErr(rows.Err())
}

// SetPrimary marks one resume primary and clears the flag on the candidate's
// others, atomically.
func (r *ResumeRepo) SetPrimary(ctx context.Context, userID, resumeID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return mapErr(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `UPDATE resumes SET is_primary = false WHERE candidate_user_id = $1`, userID); err != nil {
		return mapErr(err)
	}
	ct, err := tx.Exec(ctx,
		`UPDATE resumes SET is_primary = true WHERE id = $1 AND candidate_user_id = $2`, resumeID, userID)
	if err != nil {
		return mapErr(err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return mapErr(tx.Commit(ctx))
}

// SetParsed records the extracted data and marks the resume parsed.
func (r *ResumeRepo) SetParsed(ctx context.Context, id int64, parsed []byte) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE resumes SET parsed_data = $2, parsed_at = now() WHERE id = $1`, id, parsed)
	if err != nil {
		return mapErr(err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
