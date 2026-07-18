package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/faizan/ats/internal/domain"
)

// OfferRepo persists offer letters.
type OfferRepo struct{ pool *pgxpool.Pool }

// salary_amount is read as float8 to avoid numeric<->float scan friction; the
// range of numeric(12,2) is well within float64 precision.
const offerCols = `ol.id, ol.application_id, ol.created_by, ol.position_title, ` +
	`ol.salary_amount::float8, ol.salary_currency, ol.start_date, ol.status, ` +
	`COALESCE(ol.storage_key,''), ol.expires_at, ol.created_at, ol.updated_at`

func scanOffer(s scanner) (*domain.Offer, error) {
	var o domain.Offer
	var status string
	if err := s.Scan(&o.ID, &o.ApplicationID, &o.CreatedBy, &o.PositionTitle,
		&o.SalaryAmount, &o.SalaryCurrency, &o.StartDate, &status, &o.StorageKey,
		&o.ExpiresAt, &o.CreatedAt, &o.UpdatedAt); err != nil {
		return nil, mapErr(err)
	}
	o.Status = domain.OfferStatus(status)
	return &o, nil
}

// moneyParam formats an optional salary as text so it parses cleanly into a
// numeric column via an explicit cast (avoids float->numeric encode issues).
func moneyParam(v *float64) any {
	if v == nil {
		return nil
	}
	return fmt.Sprintf("%.2f", *v)
}

func (r *OfferRepo) Create(ctx context.Context, o *domain.Offer) error {
	const q = `
		INSERT INTO offer_letters (application_id, created_by, position_title, salary_amount,
			salary_currency, start_date, status, expires_at)
		VALUES ($1, $2, $3, $4::numeric, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`
	return mapErr(r.pool.QueryRow(ctx, q,
		o.ApplicationID, o.CreatedBy, o.PositionTitle, moneyParam(o.SalaryAmount),
		o.SalaryCurrency, o.StartDate, string(o.Status), o.ExpiresAt).
		Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt))
}

func (r *OfferRepo) GetByID(ctx context.Context, id int64) (*domain.Offer, error) {
	return scanOffer(r.pool.QueryRow(ctx, `SELECT `+offerCols+` FROM offer_letters ol WHERE ol.id = $1`, id))
}

func (r *OfferRepo) SetStatus(ctx context.Context, id int64, status domain.OfferStatus) error {
	ct, err := r.pool.Exec(ctx, `UPDATE offer_letters SET status = $2 WHERE id = $1`, id, string(status))
	if err != nil {
		return mapErr(err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *OfferRepo) SetStorageKey(ctx context.Context, id int64, key string) error {
	ct, err := r.pool.Exec(ctx, `UPDATE offer_letters SET storage_key = $2 WHERE id = $1`, id, key)
	if err != nil {
		return mapErr(err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *OfferRepo) ListByApplication(ctx context.Context, appID int64) ([]domain.Offer, error) {
	return r.list(ctx, `SELECT `+offerCols+` FROM offer_letters ol WHERE ol.application_id = $1 ORDER BY ol.created_at DESC`, appID)
}

func (r *OfferRepo) ListByCandidate(ctx context.Context, candidateID int64) ([]domain.Offer, error) {
	const q = `SELECT ` + offerCols + `
		FROM offer_letters ol
		JOIN applications a ON a.id = ol.application_id
		WHERE a.candidate_id = $1
		ORDER BY ol.created_at DESC`
	return r.list(ctx, q, candidateID)
}

func (r *OfferRepo) list(ctx context.Context, q string, arg int64) ([]domain.Offer, error) {
	rows, err := r.pool.Query(ctx, q, arg)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []domain.Offer
	for rows.Next() {
		o, err := scanOffer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, mapErr(rows.Err())
}
