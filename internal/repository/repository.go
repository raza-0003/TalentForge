// Package repository is the data-access layer over PostgreSQL (pgx).
package repository

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/faizan/ats/internal/domain"
)

// Repositories bundles every repository behind one struct.
type Repositories struct {
	Users         *UserRepo
	RefreshTokens *RefreshTokenRepo
	Profiles      *ProfileRepo
	Jobs          *JobRepo
	Applications  *ApplicationRepo
	Interviews    *InterviewRepo
	Feedback      *FeedbackRepo
	Notifications *NotificationRepo
	Resumes       *ResumeRepo
	Offers        *OfferRepo
}

// New constructs all repositories from a connection pool.
func New(pool *pgxpool.Pool) *Repositories {
	return &Repositories{
		Users:         &UserRepo{pool: pool},
		RefreshTokens: &RefreshTokenRepo{pool: pool},
		Profiles:      &ProfileRepo{pool: pool},
		Jobs:          &JobRepo{pool: pool},
		Applications:  &ApplicationRepo{pool: pool},
		Interviews:    &InterviewRepo{pool: pool},
		Feedback:      &FeedbackRepo{pool: pool},
		Notifications: &NotificationRepo{pool: pool},
		Resumes:       &ResumeRepo{pool: pool},
		Offers:        &OfferRepo{pool: pool},
	}
}

// scanner is implemented by both pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// mapErr converts pgx / PostgreSQL errors into domain sentinels.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return domain.ErrConflict
	}
	return err
}

// nullStr sends NULL for empty strings, keeping nullable columns clean.
func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nullAppStatus sends NULL or the enum's text label.
func nullAppStatus(p *domain.ApplicationStatus) any {
	if p == nil {
		return nil
	}
	return string(*p)
}
