package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/quantyralabs/idx-api/internal/domain"
)

// DomainRepo handles domains table access.
type DomainRepo struct {
	db *DB
}

func NewDomainRepo(db *DB) *DomainRepo {
	return &DomainRepo{db: db}
}

const domainSelectCols = `
	id, user_id, domain_slug, is_active, mls_dataset, allowed_mls_datasets,
	verification_status, verification_method, txt_verification_name, txt_verification_value,
	txt_verified_at, created_at, updated_at
`

func (r *DomainRepo) FindActiveBySlug(ctx context.Context, slug string) (*domain.Domain, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	row := pool.QueryRow(ctx, `
		SELECT `+domainSelectCols+`
		FROM domains
		WHERE is_active = true AND LOWER(domain_slug) = LOWER($1)
		LIMIT 1
	`, slug)
	d, err := scanDomain(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (r *DomainRepo) FindActiveForUser(ctx context.Context, userID int64, slug string) (*domain.Domain, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	row := pool.QueryRow(ctx, `
		SELECT `+domainSelectCols+`
		FROM domains
		WHERE is_active = true AND user_id = $1 AND LOWER(domain_slug) = LOWER($2)
		LIMIT 1
	`, userID, slug)
	d, err := scanDomain(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (r *DomainRepo) ListActive(ctx context.Context) ([]domain.Domain, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT `+domainSelectCols+`
		FROM domains WHERE is_active = true
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Domain
	for rows.Next() {
		d, err := scanDomain(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func scanDomain(row pgx.Row) (*domain.Domain, error) {
	var d domain.Domain
	err := row.Scan(
		&d.ID, &d.UserID, &d.DomainSlug, &d.IsActive, &d.MLSDataset, &d.AllowedMLSDatasets,
		&d.VerificationStatus, &d.VerificationMethod, &d.TXTVerificationName, &d.TXTVerificationValue,
		&d.TXTVerifiedAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
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
