package domain

import (
	"encoding/json"
	"time"
)

// Domain is a customer hostname registration.
type Domain struct {
	ID                   int64           `db:"id"`
	UserID               *int64          `db:"user_id"`
	ParentDomainID       *int64          `db:"parent_domain_id"`
	IsStaging            bool            `db:"is_staging"`
	DomainSlug           string          `db:"domain_slug"`
	IsActive             bool            `db:"is_active"`
	MLSDataset           *string         `db:"mls_dataset"`
	AllowedMLSDatasets   json.RawMessage `db:"allowed_mls_datasets"`
	VerificationStatus   string          `db:"verification_status"`
	VerificationMethod   *string         `db:"verification_method"`
	TXTVerificationName  *string         `db:"txt_verification_name"`
	TXTVerificationValue *string         `db:"txt_verification_value"`
	TXTVerifiedAt        *time.Time      `db:"txt_verified_at"`
	CreatedAt            time.Time       `db:"created_at"`
	UpdatedAt            time.Time       `db:"updated_at"`
}

func (d Domain) IsVerified() bool {
	return d.VerificationStatus == "verified" || d.VerificationStatus == "verified_ghl"
}

// User is a dashboard account.
type User struct {
	ID       int64  `db:"id"`
	Name     string `db:"name"`
	Email    string `db:"email"`
	Password string `db:"password"`
	IsAdmin  bool   `db:"is_admin"`
}

// APIToken is a bearer token (hashed at rest).
type APIToken struct {
	ID         int64      `db:"id"`
	UserID     int64      `db:"tokenable_id"`
	DomainID   *int64     `db:"domain_id"`
	Name       string     `db:"name"`
	TokenHash  string     `db:"token"`
	Abilities  *string    `db:"abilities"`
	LastUsedAt *time.Time `db:"last_used_at"`
	ExpiresAt  *time.Time `db:"expires_at"`
}
