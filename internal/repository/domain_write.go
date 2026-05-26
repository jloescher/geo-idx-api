package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// DomainCreateParams holds fields for inserting a domain row.
type DomainCreateParams struct {
	UserID               int64
	ParentDomainID       *int64
	IsStaging            bool
	DomainSlug           string
	MLSDataset           string
	AllowedMLSDatasets   []byte // JSON array
	TXTVerificationName  string
	TXTVerificationValue string
}

// CreateDomain inserts a domain within a transaction.
func CreateDomain(ctx context.Context, tx pgx.Tx, p DomainCreateParams) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO domains (user_id, parent_domain_id, is_staging, domain_slug, mls_dataset, allowed_mls_datasets, verification_status,
			txt_verification_name, txt_verification_value, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, 'pending', $7, $8, NOW(), NOW())
		RETURNING id
	`, p.UserID, p.ParentDomainID, p.IsStaging, p.DomainSlug, p.MLSDataset, string(p.AllowedMLSDatasets), p.TXTVerificationName, p.TXTVerificationValue).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert domain %s: %w", p.DomainSlug, err)
	}
	return id, nil
}

// DeleteDomainBundle removes a production domain; CASCADE deletes staging child and domain-scoped tokens.
func DeleteDomainBundle(ctx context.Context, db *DB, userID, prodDomainID int64) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var isStaging bool
	var parentID *int64
	err = tx.QueryRow(ctx, `
		SELECT is_staging, parent_domain_id FROM domains WHERE id = $1 AND user_id = $2
	`, prodDomainID, userID).Scan(&isStaging, &parentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("domain not found")
	}
	if err != nil {
		return err
	}
	if isStaging || parentID != nil {
		return fmt.Errorf("delete the production domain, not staging")
	}

	tag, err := tx.Exec(ctx, `DELETE FROM domains WHERE id = $1 AND user_id = $2`, prodDomainID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("domain not found")
	}
	return tx.Commit(ctx)
}
