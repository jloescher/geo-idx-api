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
		SELECT id, tokenable_id, name, token, abilities, last_used_at, expires_at
		FROM personal_access_tokens
		WHERE tokenable_type = 'App\\Models\\User' AND token = $1
		LIMIT 1
	`, hash).Scan(&tok.ID, &tok.UserID, &tok.Name, &tok.TokenHash, &tok.Abilities, &tok.LastUsedAt, &tok.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		// Legacy Sanctum: token id|secret — re-issue required; not supported in Go path
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
	return r.createToken(ctx, r.db.Pool, userID, name, abilities)
}

// CreateWithTx mints a token inside an existing transaction.
func (r *TokenRepo) CreateWithTx(ctx context.Context, tx pgx.Tx, userID int64, name string, abilities []string) (plain string, err error) {
	return r.createToken(ctx, tx, userID, name, abilities)
}

type execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func (r *TokenRepo) createToken(ctx context.Context, db execer, userID int64, name string, abilities []string) (plain string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	plain = "idx_" + hex.EncodeToString(b)
	abil := `["` + strings.Join(abilities, `","`) + `"]`
	_, err = db.Exec(ctx, `
		INSERT INTO personal_access_tokens (tokenable_type, tokenable_id, name, token, abilities, created_at, updated_at)
		VALUES ('App\\Models\\User', $1, $2, $3, $4, NOW(), NOW())
	`, userID, name, HashToken(plain), abil)
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
