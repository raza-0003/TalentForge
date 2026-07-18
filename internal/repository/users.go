package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/faizan/ats/internal/domain"
)

// UserRepo persists user accounts.
type UserRepo struct{ pool *pgxpool.Pool }

const userCols = `id, email, password_hash, full_name, role, is_active, created_at, updated_at`

func scanUser(s scanner) (*domain.User, error) {
	var u domain.User
	var role string
	if err := s.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &role,
		&u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, mapErr(err)
	}
	u.Role = domain.Role(role)
	return &u, nil
}

// Create inserts a new user and fills generated fields.
func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	const q = `
		INSERT INTO users (email, password_hash, full_name, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, is_active, created_at, updated_at`
	return mapErr(r.pool.QueryRow(ctx, q, u.Email, u.PasswordHash, u.FullName, string(u.Role)).
		Scan(&u.ID, &u.IsActive, &u.CreatedAt, &u.UpdatedAt))
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return scanUser(r.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE email = $1`, email))
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	return scanUser(r.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id = $1`, id))
}

func (r *UserRepo) SetActive(ctx context.Context, id int64, active bool) error {
	ct, err := r.pool.Exec(ctx, `UPDATE users SET is_active = $2 WHERE id = $1`, id, active)
	if err != nil {
		return mapErr(err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepo) ListByRole(ctx context.Context, role domain.Role, limit, offset int) ([]domain.User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+userCols+` FROM users WHERE role = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		string(role), limit, offset)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, mapErr(rows.Err())
}

// RefreshTokenRepo persists hashed, revocable refresh tokens.
type RefreshTokenRepo struct{ pool *pgxpool.Pool }

func (r *RefreshTokenRepo) Store(ctx context.Context, userID int64, hash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, hash, expiresAt)
	return mapErr(err)
}

// Consume atomically validates and revokes a refresh token (single-use
// rotation), returning the owning user id. Returns ErrNotFound if the token is
// unknown, expired, or already used.
func (r *RefreshTokenRepo) Consume(ctx context.Context, hash string) (int64, error) {
	const q = `
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now()
		RETURNING user_id`
	var userID int64
	if err := r.pool.QueryRow(ctx, q, hash).Scan(&userID); err != nil {
		return 0, mapErr(err)
	}
	return userID, nil
}

func (r *RefreshTokenRepo) Revoke(ctx context.Context, hash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL`, hash)
	return mapErr(err)
}
