package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/quantyralabs/idx-api/internal/domain"
)

// DomainRepo handles domains table access.
type DomainRepo struct {
	db *DB
}

func NewDomainRepo(db *DB) *DomainRepo {
	return &DomainRepo{db: db}
}

func (r *DomainRepo) FindActiveBySlug(ctx context.Context, slug string) (*domain.Domain, error) {
	var d domain.Domain
	err := r.db.SQLX.GetContext(ctx, &d, `
		SELECT id, user_id, domain_slug, is_active, mls_dataset, allowed_mls_datasets,
		       verification_status, verification_method, txt_verification_name, txt_verification_value,
		       txt_verified_at, created_at, updated_at
		FROM domains
		WHERE is_active = true AND LOWER(domain_slug) = LOWER($1)
		LIMIT 1
	`, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DomainRepo) FindActiveForUser(ctx context.Context, userID int64, slug string) (*domain.Domain, error) {
	var d domain.Domain
	err := r.db.SQLX.GetContext(ctx, &d, `
		SELECT id, user_id, domain_slug, is_active, mls_dataset, allowed_mls_datasets,
		       verification_status, verification_method, txt_verification_name, txt_verification_value,
		       txt_verified_at, created_at, updated_at
		FROM domains
		WHERE is_active = true AND user_id = $1 AND LOWER(domain_slug) = LOWER($2)
		LIMIT 1
	`, userID, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DomainRepo) ListActive(ctx context.Context) ([]domain.Domain, error) {
	var rows []domain.Domain
	err := r.db.SQLX.SelectContext(ctx, &rows, `
		SELECT id, user_id, domain_slug, is_active, mls_dataset, allowed_mls_datasets,
		       verification_status, verification_method, txt_verification_name, txt_verification_value,
		       txt_verified_at, created_at, updated_at
		FROM domains WHERE is_active = true
	`)
	return rows, err
}

func (r *DomainRepo) AllowedDatasets(d *domain.Domain) []string {
	if d == nil || len(d.AllowedMLSDatasets) == 0 {
		return nil
	}
	var raw []string
	if err := jsonUnmarshal(d.AllowedMLSDatasets, &raw); err != nil {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
