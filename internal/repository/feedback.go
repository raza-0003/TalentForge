package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/faizan/ats/internal/domain"
)

// FeedbackRepo persists interview feedback.
type FeedbackRepo struct{ pool *pgxpool.Pool }

func (r *FeedbackRepo) Create(ctx context.Context, f *domain.Feedback) error {
	const q = `
		INSERT INTO interview_feedback (interview_id, author_id, rating, recommendation, strengths, weaknesses, comments)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`
	return mapErr(r.pool.QueryRow(ctx, q,
		f.InterviewID, f.AuthorID, f.Rating, string(f.Recommendation),
		nullStr(f.Strengths), nullStr(f.Weaknesses), nullStr(f.Comments)).
		Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt))
}

func (r *FeedbackRepo) GetByInterview(ctx context.Context, interviewID int64) (*domain.Feedback, error) {
	const q = `
		SELECT id, interview_id, author_id, rating, recommendation,
		       COALESCE(strengths,''), COALESCE(weaknesses,''), COALESCE(comments,''), created_at, updated_at
		FROM interview_feedback WHERE interview_id = $1`
	var f domain.Feedback
	var rec string
	err := r.pool.QueryRow(ctx, q, interviewID).Scan(
		&f.ID, &f.InterviewID, &f.AuthorID, &f.Rating, &rec,
		&f.Strengths, &f.Weaknesses, &f.Comments, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	f.Recommendation = domain.Recommendation(rec)
	return &f, nil
}
