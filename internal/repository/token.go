package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/quantyralabs/idx-api/internal/domain"
)

// TokenRepo manages API tokens (Go format: sha256 hex in token column).
type TokenRepo struct {
	db *DB
}

// TokenMeta is dashboard-safe token metadata (no secret).
type TokenMeta struct {
	ID        int64
	DomainID  int64
	Name      string
	LastUsed  string
	NeverUsed bool
}

func NewTokenRepo(db *DB) *TokenRepo {
	return &TokenRepo{db: db}
}

func HashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

func (r *TokenRepo) FindByPlaintext(ctx context.Context, plain string) (*domain.APIToken, *domain.User, error) {
	hash := HashToken(plain)
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, nil, err
	}
	var tok domain.APIToken
	err = pool.QueryRow(ctx, `
		SELECT id, tokenable_id, domain_id, name, token, abilities, last_used_at, expires_at
		FROM personal_access_tokens
		WHERE tokenable_type = 'App\\Models\\User' AND token = $1
		LIMIT 1
	`, hash).Scan(&tok.ID, &tok.UserID, &tok.DomainID, &tok.Name, &tok.TokenHash, &tok.Abilities, &tok.LastUsedAt, &tok.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if tok.ExpiresAt != nil && tok.ExpiresAt.Before(time.Now()) {
		return nil, nil, nil
	}
	var user domain.User
	err = pool.QueryRow(ctx, `SELECT id, name, email, password, is_admin FROM users WHERE id = $1`, tok.UserID).
		Scan(&user.ID, &user.Name, &user.Email, &user.Password, &user.IsAdmin)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	_, _ = r.db.Pool.Exec(ctx, `UPDATE personal_access_tokens SET last_used_at = NOW() WHERE id = $1`, tok.ID)
	return &tok, &user, nil
}

func (r *TokenRepo) HasAbility(tok *domain.APIToken, ability string) bool {
	if tok.Abilities == nil {
		return false
	}
	s := *tok.Abilities
	return strings.Contains(s, ability)
}

func (r *TokenRepo) Create(ctx context.Context, userID int64, name string, abilities []string) (plain string, err error) {
	return r.createToken(ctx, r.db.Pool, userID, 0, name, abilities)
}

// CreateForDomain mints a token scoped to a domain row.
func (r *TokenRepo) CreateForDomain(ctx context.Context, userID, domainID int64, name string, abilities []string) (plain string, err error) {
	return r.createToken(ctx, r.db.Pool, userID, domainID, name, abilities)
}

// CreateForDomainWithTx mints a domain-scoped token inside a transaction.
func (r *TokenRepo) CreateForDomainWithTx(ctx context.Context, tx pgx.Tx, userID, domainID int64, name string, abilities []string) (plain string, err error) {
	return r.createToken(ctx, tx, userID, domainID, name, abilities)
}

// CreateWithTx mints a token inside an existing transaction (legacy, no domain).
func (r *TokenRepo) CreateWithTx(ctx context.Context, tx pgx.Tx, userID int64, name string, abilities []string) (plain string, err error) {
	return r.createToken(ctx, tx, userID, 0, name, abilities)
}

// CreateAdminToken creates a global admin token (not tied to any domain).
// Only call this after verifying the user is an admin.
func (r *TokenRepo) CreateAdminToken(ctx context.Context, userID int64, name string, abilities []string) (plain string, err error) {
	// Admin tokens are created with domain_id = NULL and special handling.
	// We use the same table but mark them via abilities or a future column.
	// For now, we ensure "admin" ability is present and create without domain.
	if !containsAbility(abilities, "admin") {
		abilities = append(abilities, "admin")
	}
	return r.createToken(ctx, r.db.Pool, userID, 0, name, abilities)
}

func containsAbility(abilities []string, ability string) bool {
	for _, a := range abilities {
		if a == ability {
			return true
		}
	}
	return false
}

// RegenerateForDomain replaces the token for a domain atomically.
func (r *TokenRepo) RegenerateForDomain(ctx context.Context, userID, domainID int64, name string, abilities []string) (plain string, err error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var owner int64
	err = tx.QueryRow(ctx, `SELECT user_id FROM domains WHERE id = $1`, domainID).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("domain not found")
	}
	if err != nil {
		return "", err
	}
	if owner != userID {
		return "", fmt.Errorf("domain not found")
	}

	_, err = tx.Exec(ctx, `
		DELETE FROM personal_access_tokens
		WHERE tokenable_type = 'App\\Models\\User' AND tokenable_id = $1 AND domain_id = $2
	`, userID, domainID)
	if err != nil {
		return "", err
	}

	plain, err = r.createToken(ctx, tx, userID, domainID, name, abilities)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return plain, nil
}

// FindByDomain returns token metadata for a user's domain.
func (r *TokenRepo) FindByDomain(ctx context.Context, userID, domainID int64) (*TokenMeta, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	var t TokenMeta
	var lastUsed *time.Time
	err = pool.QueryRow(ctx, `
		SELECT id, domain_id, name, last_used_at
		FROM personal_access_tokens
		WHERE tokenable_type = 'App\\Models\\User' AND tokenable_id = $1 AND domain_id = $2
		LIMIT 1
	`, userID, domainID).Scan(&t.ID, &t.DomainID, &t.Name, &lastUsed)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if lastUsed != nil {
		t.LastUsed = lastUsed.Format(time.RFC3339)
	} else {
		t.NeverUsed = true
	}
	return &t, nil
}

type execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func (r *TokenRepo) createToken(ctx context.Context, db execer, userID, domainID int64, name string, abilities []string) (plain string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	plain = "idx_" + hex.EncodeToString(b)
	abil := `["` + strings.Join(abilities, `","`) + `"]`
	var domainArg any
	if domainID > 0 {
		domainArg = domainID
	}
	_, err = db.Exec(ctx, `
		INSERT INTO personal_access_tokens (tokenable_type, tokenable_id, domain_id, name, token, abilities, created_at, updated_at)
		VALUES ('App\\Models\\User', $1, $2, $3, $4, $5, NOW(), NOW())
	`, userID, domainArg, name, HashToken(plain), abil)
	return plain, err
}

func (r *TokenRepo) Revoke(ctx context.Context, userID, tokenID int64) error {
	tag, err := r.db.Pool.Exec(ctx, `
		DELETE FROM personal_access_tokens WHERE id = $1 AND tokenable_id = $2 AND tokenable_type = 'App\\Models\\User'
	`, tokenID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("token not found")
	}
	return nil
}

// ListAdminTokensForUser returns admin tokens created by the given user.
func (r *TokenRepo) ListAdminTokensForUser(ctx context.Context, userID int64) ([]TokenMeta, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, name, last_used_at
		FROM personal_access_tokens
		WHERE tokenable_type = 'App\\Models\\User'
		  AND tokenable_id = $1
		  AND abilities LIKE '%admin%'
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []TokenMeta
	for rows.Next() {
		var t TokenMeta
		var lastUsed *time.Time
		if err := rows.Scan(&t.ID, &t.Name, &lastUsed); err != nil {
			return nil, err
		}
		if lastUsed != nil {
			t.LastUsed = lastUsed.Format("2006-01-02 15:04")
		} else {
			t.NeverUsed = true
		}
		tokens = append(tokens, t)
	}
	return tokens, nil
}
