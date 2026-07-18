package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/faizan/ats/internal/domain"
)

// ApplicationRepo persists applications and their timeline events.
type ApplicationRepo struct{ pool *pgxpool.Pool }

const appCols = `id, job_id, candidate_id, resume_id, status, COALESCE(cover_letter,''), created_at, updated_at`

func scanApp(s scanner) (*domain.Application, error) {
	var a domain.Application
	var status string
	if err := s.Scan(&a.ID, &a.JobID, &a.CandidateID, &a.ResumeID, &status,
		&a.CoverLetter, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, mapErr(err)
	}
	a.Status = domain.ApplicationStatus(status)
	return &a, nil
}

func (r *ApplicationRepo) Create(ctx context.Context, a *domain.Application) error {
	const q = `
		INSERT INTO applications (job_id, candidate_id, resume_id, status, cover_letter)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	return mapErr(r.pool.QueryRow(ctx, q,
		a.JobID, a.CandidateID, a.ResumeID, string(a.Status), nullStr(a.CoverLetter)).
		Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt))
}

func (r *ApplicationRepo) GetByID(ctx context.Context, id int64) (*domain.Application, error) {
	return scanApp(r.pool.QueryRow(ctx, `SELECT `+appCols+` FROM applications WHERE id = $1`, id))
}

func (r *ApplicationRepo) UpdateStatus(ctx context.Context, id int64, status domain.ApplicationStatus) error {
	ct, err := r.pool.Exec(ctx, `UPDATE applications SET status = $2 WHERE id = $1`, id, string(status))
	if err != nil {
		return mapErr(err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ListByJob returns applications for a job (recruiter view), with candidate
// names, optionally filtered by status ("" means all).
func (r *ApplicationRepo) ListByJob(ctx context.Context, jobID int64, status domain.ApplicationStatus, limit, offset int) ([]domain.Application, error) {
	const q = `
		SELECT a.id, a.job_id, a.candidate_id, a.resume_id, a.status,
		       COALESCE(a.cover_letter,''), a.created_at, a.updated_at, u.full_name
		FROM applications a JOIN users u ON u.id = a.candidate_id
		WHERE a.job_id = $1 AND ($2 = '' OR a.status::text = $2)
		ORDER BY a.created_at DESC LIMIT $3 OFFSET $4`
	rows, err := r.pool.Query(ctx, q, jobID, string(status), limit, offset)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []domain.Application
	for rows.Next() {
		var a domain.Application
		var st string
		if err := rows.Scan(&a.ID, &a.JobID, &a.CandidateID, &a.ResumeID, &st,
			&a.CoverLetter, &a.CreatedAt, &a.UpdatedAt, &a.CandidateName); err != nil {
			return nil, mapErr(err)
		}
		a.Status = domain.ApplicationStatus(st)
		out = append(out, a)
	}
	return out, mapErr(rows.Err())
}

// ListByCandidate returns a candidate's own applications, with job titles.
func (r *ApplicationRepo) ListByCandidate(ctx context.Context, candidateID int64, limit, offset int) ([]domain.Application, error) {
	const q = `
		SELECT a.id, a.job_id, a.candidate_id, a.resume_id, a.status,
		       COALESCE(a.cover_letter,''), a.created_at, a.updated_at, j.title
		FROM applications a JOIN jobs j ON j.id = a.job_id
		WHERE a.candidate_id = $1
		ORDER BY a.created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, q, candidateID, limit, offset)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []domain.Application
	for rows.Next() {
		var a domain.Application
		var st string
		if err := rows.Scan(&a.ID, &a.JobID, &a.CandidateID, &a.ResumeID, &st,
			&a.CoverLetter, &a.CreatedAt, &a.UpdatedAt, &a.JobTitle); err != nil {
			return nil, mapErr(err)
		}
		a.Status = domain.ApplicationStatus(st)
		out = append(out, a)
	}
	return out, mapErr(rows.Err())
}

// AddEvent appends an entry to an application's timeline.
func (r *ApplicationRepo) AddEvent(ctx context.Context, e *domain.ApplicationEvent) error {
	const q = `
		INSERT INTO application_events (application_id, actor_id, event_type, from_status, to_status, note)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`
	return mapErr(r.pool.QueryRow(ctx, q,
		e.ApplicationID, e.ActorID, e.EventType,
		nullAppStatus(e.FromStatus), nullAppStatus(e.ToStatus), nullStr(e.Note)).
		Scan(&e.ID, &e.CreatedAt))
}

// ListEvents returns an application's timeline in chronological order.
func (r *ApplicationRepo) ListEvents(ctx context.Context, appID int64) ([]domain.ApplicationEvent, error) {
	const q = `
		SELECT id, application_id, actor_id, event_type, from_status, to_status, COALESCE(note,''), created_at
		FROM application_events WHERE application_id = $1 ORDER BY created_at`
	rows, err := r.pool.Query(ctx, q, appID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []domain.ApplicationEvent
	for rows.Next() {
		var e domain.ApplicationEvent
		var from, to *string
		if err := rows.Scan(&e.ID, &e.ApplicationID, &e.ActorID, &e.EventType,
			&from, &to, &e.Note, &e.CreatedAt); err != nil {
			return nil, mapErr(err)
		}
		if from != nil {
			fs := domain.ApplicationStatus(*from)
			e.FromStatus = &fs
		}
		if to != nil {
			ts := domain.ApplicationStatus(*to)
			e.ToStatus = &ts
		}
		out = append(out, e)
	}
	return out, mapErr(rows.Err())
}

// CountByStatus returns application counts grouped by status (for analytics).
func (r *ApplicationRepo) CountByStatus(ctx context.Context) (map[string]int, error) {
	rows, err := r.pool.Query(ctx, `SELECT status::text, count(*) FROM applications GROUP BY status`)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var s string
		var n int
		if err := rows.Scan(&s, &n); err != nil {
			return nil, mapErr(err)
		}
		out[s] = n
	}
	return out, mapErr(rows.Err())
}
