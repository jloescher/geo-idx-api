package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// DomainCreateParams holds fields for inserting a domain row.
type DomainCreateParams struct {
	UserID               int64
	DomainSlug           string
	MLSDataset           string
	AllowedMLSDatasets   []byte // JSON array
	TXTVerificationName  string
	TXTVerificationValue string
}

// CreateDomain inserts a domain within a transaction.
// Revenue impact: atomic domain rows keep pilot onboarding from half-provisioned DNS states.
func CreateDomain(ctx context.Context, tx pgx.Tx, p DomainCreateParams) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO domains (user_id, domain_slug, mls_dataset, allowed_mls_datasets, verification_status,
			txt_verification_name, txt_verification_value, created_at, updated_at)
		VALUES ($1, $2, $3, $4::jsonb, 'pending', $5, $6, NOW(), NOW())
		RETURNING id
	`, p.UserID, p.DomainSlug, p.MLSDataset, string(p.AllowedMLSDatasets), p.TXTVerificationName, p.TXTVerificationValue).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert domain %s: %w", p.DomainSlug, err)
	}
	return id, nil
}
