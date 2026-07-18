package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/faizan/ats/internal/domain"
)

// JobRepo persists job postings.
type JobRepo struct{ pool *pgxpool.Pool }

const jobCols = `id, created_by, title, description, COALESCE(department,''), ` +
	`COALESCE(location,''), COALESCE(employment_type,''), min_experience, status, created_at, updated_at`

func scanJob(s scanner) (*domain.Job, error) {
	var j domain.Job
	var status string
	if err := s.Scan(&j.ID, &j.CreatedBy, &j.Title, &j.Description, &j.Department,
		&j.Location, &j.EmploymentType, &j.MinExperience, &status,
		&j.CreatedAt, &j.UpdatedAt); err != nil {
		return nil, mapErr(err)
	}
	j.Status = domain.JobStatus(status)
	return &j, nil
}

func (r *JobRepo) Create(ctx context.Context, j *domain.Job) error {
	const q = `
		INSERT INTO jobs (created_by, title, description, department, location, employment_type, min_experience, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`
	return mapErr(r.pool.QueryRow(ctx, q,
		j.CreatedBy, j.Title, j.Description, nullStr(j.Department), nullStr(j.Location),
		nullStr(j.EmploymentType), j.MinExperience, string(j.Status)).
		Scan(&j.ID, &j.CreatedAt, &j.UpdatedAt))
}

func (r *JobRepo) GetByID(ctx context.Context, id int64) (*domain.Job, error) {
	return scanJob(r.pool.QueryRow(ctx, `SELECT `+jobCols+` FROM jobs WHERE id = $1`, id))
}

func (r *JobRepo) Update(ctx context.Context, j *domain.Job) error {
	const q = `
		UPDATE jobs SET title = $2, description = $3, department = $4, location = $5,
			employment_type = $6, min_experience = $7, status = $8
		WHERE id = $1`
	ct, err := r.pool.Exec(ctx, q, j.ID, j.Title, j.Description, nullStr(j.Department),
		nullStr(j.Location), nullStr(j.EmploymentType), j.MinExperience, string(j.Status))
	if err != nil {
		return mapErr(err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *JobRepo) SetStatus(ctx context.Context, id int64, status domain.JobStatus) error {
	ct, err := r.pool.Exec(ctx, `UPDATE jobs SET status = $2 WHERE id = $1`, id, string(status))
	if err != nil {
		return mapErr(err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// List returns jobs, optionally filtered by status and a title/description search.
func (r *JobRepo) List(ctx context.Context, status domain.JobStatus, search string, limit, offset int) ([]domain.Job, error) {
	var (
		conds []string
		args  []any
	)
	if status != "" {
		args = append(args, string(status))
		conds = append(conds, fmt.Sprintf("status = $%d", len(args)))
	}
	if search != "" {
		args = append(args, "%"+search+"%")
		conds = append(conds, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", len(args), len(args)))
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	args = append(args, limit, offset)
	q := fmt.Sprintf(`SELECT %s FROM jobs %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		jobCols, where, len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []domain.Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *j)
	}
	return out, mapErr(rows.Err())
}

// CountByStatus returns job counts grouped by status (for analytics).
func (r *JobRepo) CountByStatus(ctx context.Context) (map[string]int, error) {
	rows, err := r.pool.Query(ctx, `SELECT status::text, count(*) FROM jobs GROUP BY status`)
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
