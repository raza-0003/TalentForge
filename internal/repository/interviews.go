package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/faizan/ats/internal/domain"
)

// InterviewRepo persists interviews.
type InterviewRepo struct{ pool *pgxpool.Pool }

const ivCols = `id, application_id, interviewer_id, created_by, scheduled_at, ` +
	`duration_minutes, mode, COALESCE(location,''), status, created_at, updated_at`

func scanInterview(s scanner) (*domain.Interview, error) {
	var iv domain.Interview
	var mode, status string
	if err := s.Scan(&iv.ID, &iv.ApplicationID, &iv.InterviewerID, &iv.CreatedBy,
		&iv.ScheduledAt, &iv.DurationMinutes, &mode, &iv.Location, &status,
		&iv.CreatedAt, &iv.UpdatedAt); err != nil {
		return nil, mapErr(err)
	}
	iv.Mode = domain.InterviewMode(mode)
	iv.Status = domain.InterviewStatus(status)
	return &iv, nil
}

func (r *InterviewRepo) Create(ctx context.Context, iv *domain.Interview) error {
	const q = `
		INSERT INTO interviews (application_id, interviewer_id, created_by, scheduled_at,
			duration_minutes, mode, location, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`
	return mapErr(r.pool.QueryRow(ctx, q,
		iv.ApplicationID, iv.InterviewerID, iv.CreatedBy, iv.ScheduledAt,
		iv.DurationMinutes, string(iv.Mode), nullStr(iv.Location), string(iv.Status)).
		Scan(&iv.ID, &iv.CreatedAt, &iv.UpdatedAt))
}

func (r *InterviewRepo) GetByID(ctx context.Context, id int64) (*domain.Interview, error) {
	return scanInterview(r.pool.QueryRow(ctx, `SELECT `+ivCols+` FROM interviews WHERE id = $1`, id))
}

func (r *InterviewRepo) ListByApplication(ctx context.Context, appID int64) ([]domain.Interview, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+ivCols+` FROM interviews WHERE application_id = $1 ORDER BY scheduled_at`, appID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []domain.Interview
	for rows.Next() {
		iv, err := scanInterview(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *iv)
	}
	return out, mapErr(rows.Err())
}

func (r *InterviewRepo) Update(ctx context.Context, id int64, scheduledAt time.Time, duration int,
	mode domain.InterviewMode, location string, status domain.InterviewStatus) error {
	const q = `
		UPDATE interviews SET scheduled_at = $2, duration_minutes = $3, mode = $4,
			location = $5, status = $6
		WHERE id = $1`
	ct, err := r.pool.Exec(ctx, q, id, scheduledAt, duration, string(mode), nullStr(location), string(status))
	if err != nil {
		return mapErr(err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
