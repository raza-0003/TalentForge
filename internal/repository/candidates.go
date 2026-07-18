package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/faizan/ats/internal/domain"
)

// ProfileRepo persists candidate profiles.
type ProfileRepo struct{ pool *pgxpool.Pool }

func (r *ProfileRepo) Get(ctx context.Context, userID int64) (*domain.CandidateProfile, error) {
	const q = `
		SELECT id, user_id, COALESCE(phone,''), COALESCE(headline,''), COALESCE(location,''),
		       links, skills, created_at, updated_at
		FROM candidate_profiles WHERE user_id = $1`
	var p domain.CandidateProfile
	err := r.pool.QueryRow(ctx, q, userID).Scan(
		&p.ID, &p.UserID, &p.Phone, &p.Headline, &p.Location,
		&p.Links, &p.Skills, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &p, nil
}

// Upsert creates or replaces the profile for a candidate user.
func (r *ProfileRepo) Upsert(ctx context.Context, p *domain.CandidateProfile) error {
	const q = `
		INSERT INTO candidate_profiles (user_id, phone, headline, location, links, skills)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
			phone    = EXCLUDED.phone,
			headline = EXCLUDED.headline,
			location = EXCLUDED.location,
			links    = EXCLUDED.links,
			skills   = EXCLUDED.skills
		RETURNING id, created_at, updated_at`
	return mapErr(r.pool.QueryRow(ctx, q,
		p.UserID, nullStr(p.Phone), nullStr(p.Headline), nullStr(p.Location), p.Links, p.Skills).
		Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt))
}

// Search finds active candidates by skills and/or a name/headline query.
// When skills are given it uses the GIN index on candidate_profiles.skills:
// matchAll uses "contains all" (@>), otherwise "overlaps any" (&&).
func (r *ProfileRepo) Search(ctx context.Context, skills []string, query string, matchAll bool, limit, offset int) ([]domain.CandidateSearchResult, error) {
	conds := []string{"u.is_active", "u.role = 'candidate'"}
	var args []any

	if len(skills) > 0 {
		args = append(args, skills)
		op := "&&"
		if matchAll {
			op = "@>"
		}
		conds = append(conds, fmt.Sprintf("cp.skills %s $%d::text[]", op, len(args)))
	}
	if query != "" {
		args = append(args, "%"+query+"%")
		conds = append(conds, fmt.Sprintf("(u.full_name ILIKE $%d OR COALESCE(cp.headline,'') ILIKE $%d)", len(args), len(args)))
	}
	args = append(args, limit, offset)

	q := fmt.Sprintf(`
		SELECT u.id, u.full_name, u.email, COALESCE(cp.headline,''), COALESCE(cp.location,''), cp.skills
		FROM candidate_profiles cp
		JOIN users u ON u.id = cp.user_id
		WHERE %s
		ORDER BY u.full_name
		LIMIT $%d OFFSET $%d`,
		strings.Join(conds, " AND "), len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []domain.CandidateSearchResult
	for rows.Next() {
		var c domain.CandidateSearchResult
		if err := rows.Scan(&c.UserID, &c.FullName, &c.Email, &c.Headline, &c.Location, &c.Skills); err != nil {
			return nil, mapErr(err)
		}
		out = append(out, c)
	}
	return out, mapErr(rows.Err())
}
