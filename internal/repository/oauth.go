package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OAuthAuthorizationCode represents a short-lived authorization code.
type OAuthAuthorizationCode struct {
	Code                string    `db:"code"`
	ClientID            string    `db:"client_id"`
	UserID              int64     `db:"user_id"`
	RedirectURI         string    `db:"redirect_uri"`
	Scope               string    `db:"scope"`
	CodeChallenge       *string   `db:"code_challenge"`
	CodeChallengeMethod *string   `db:"code_challenge_method"`
	ExpiresAt           time.Time `db:"expires_at"`
	Used                bool      `db:"used"`
	CreatedAt           time.Time `db:"created_at"`
}

// OAuthAccessToken represents an issued access token.
type OAuthAccessToken struct {
	TokenHash         string    `db:"token_hash"`
	ClientID          string    `db:"client_id"`
	UserID            int64     `db:"user_id"`
	Scope             string    `db:"scope"`
	GrantedMCPKeyIDs  []int64   `db:"granted_mcp_key_ids"`
	ExpiresAt         time.Time `db:"expires_at"`
	CreatedAt         time.Time `db:"created_at"`
}

// OAuthRefreshToken represents a long-lived refresh token.
type OAuthRefreshToken struct {
	TokenHash        string    `db:"token_hash"`
	ClientID         string    `db:"client_id"`
	UserID           int64     `db:"user_id"`
	Scope            string    `db:"scope"`
	GrantedMCPKeyIDs []int64   `db:"granted_mcp_key_ids"`
	ExpiresAt        time.Time `db:"expires_at"`
	RevokedAt        *time.Time `db:"revoked_at"`
	CreatedAt        time.Time `db:"created_at"`
}

type OAuthConsent struct {
	ID               int64     `db:"id"`
	UserID           int64     `db:"user_id"`
	ClientID         string    `db:"client_id"`
	Scope            string    `db:"scope"`
	GrantedMCPKeyIDs []int64   `db:"granted_mcp_key_ids"`
	ConsentedAt      time.Time `db:"consented_at"`
	RevokedAt        *time.Time `db:"revoked_at"`
}

// OAuthRepo handles OAuth-related persistence.
type OAuthRepo struct {
	pool *pgxpool.Pool
}

// NewOAuthRepo creates a new OAuth repository.
func NewOAuthRepo(db *DB) *OAuthRepo {
	return &OAuthRepo{pool: db.Pool}
}

// CreateAuthorizationCode stores a new authorization code.
func (r *OAuthRepo) CreateAuthorizationCode(ctx context.Context, code *OAuthAuthorizationCode) error {
	query := `
		INSERT INTO oauth_authorization_codes 
		(code, client_id, user_id, redirect_uri, scope, code_challenge, code_challenge_method, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, query,
		code.Code, code.ClientID, code.UserID, code.RedirectURI, code.Scope,
		code.CodeChallenge, code.CodeChallengeMethod, code.ExpiresAt,
	)
	return err
}

// ConsumeAuthorizationCode marks a code as used and returns it if valid.
func (r *OAuthRepo) ConsumeAuthorizationCode(ctx context.Context, code string) (*OAuthAuthorizationCode, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := `
		SELECT code, client_id, user_id, redirect_uri, scope, code_challenge, code_challenge_method, expires_at, used
		FROM oauth_authorization_codes 
		WHERE code = $1 FOR UPDATE
	`

	var ac OAuthAuthorizationCode
	err = tx.QueryRow(ctx, query, code).Scan(
		&ac.Code, &ac.ClientID, &ac.UserID, &ac.RedirectURI, &ac.Scope,
		&ac.CodeChallenge, &ac.CodeChallengeMethod, &ac.ExpiresAt, &ac.Used,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if ac.Used || time.Now().After(ac.ExpiresAt) {
		return nil, nil
	}

	// Mark as used
	_, err = tx.Exec(ctx, `UPDATE oauth_authorization_codes SET used = true WHERE code = $1`, code)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &ac, nil
}

func encodeGrantedMCPKeyIDs(ids []int64) ([]byte, error) {
	if ids == nil {
		ids = []int64{}
	}
	return json.Marshal(ids)
}

func decodeGrantedMCPKeyIDs(raw []byte, dest *[]int64) error {
	if len(raw) == 0 {
		*dest = nil
		return nil
	}
	return json.Unmarshal(raw, dest)
}

// CreateAccessToken stores a new access token (we store the hash).
func (r *OAuthRepo) CreateAccessToken(ctx context.Context, token *OAuthAccessToken) error {
	idsJSON, err := encodeGrantedMCPKeyIDs(token.GrantedMCPKeyIDs)
	if err != nil {
		return fmt.Errorf("marshal granted_mcp_key_ids: %w", err)
	}
	query := `
		INSERT INTO oauth_access_tokens 
		(token_hash, client_id, user_id, scope, granted_mcp_key_ids, expires_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6)
	`
	_, err = r.pool.Exec(ctx, query,
		token.TokenHash, token.ClientID, token.UserID, token.Scope,
		string(idsJSON), token.ExpiresAt,
	)
	return err
}

// FindAccessTokenByHash retrieves a non-expired access token by its hash.
func (r *OAuthRepo) FindAccessTokenByHash(ctx context.Context, tokenHash string) (*OAuthAccessToken, error) {
	query := `
		SELECT token_hash, client_id, user_id, scope, granted_mcp_key_ids, expires_at, created_at
		FROM oauth_access_tokens 
		WHERE token_hash = $1 AND expires_at > NOW()
		LIMIT 1
	`

	var t OAuthAccessToken
	var idsJSON []byte
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&t.TokenHash, &t.ClientID, &t.UserID, &t.Scope, &idsJSON,
		&t.ExpiresAt, &t.CreatedAt,
	)
	if err == nil {
		if err := decodeGrantedMCPKeyIDs(idsJSON, &t.GrantedMCPKeyIDs); err != nil {
			return nil, fmt.Errorf("decode granted_mcp_key_ids: %w", err)
		}
	}
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// RevokeAccessToken deletes an access token (revokes it).
func (r *OAuthRepo) RevokeAccessToken(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM oauth_access_tokens WHERE token_hash = $1`, tokenHash)
	return err
}

// ListActiveAccessTokensByClient returns currently valid access tokens for a given client.
func (r *OAuthRepo) ListActiveAccessTokensByClient(ctx context.Context, clientID string) ([]OAuthAccessToken, error) {
	query := `
		SELECT token_hash, client_id, user_id, scope, granted_mcp_key_ids, expires_at, created_at
		FROM oauth_access_tokens
		WHERE client_id = $1 AND expires_at > NOW()
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []OAuthAccessToken
	for rows.Next() {
		t, err := scanOAuthAccessToken(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, nil
}

// ListAllActiveAccessTokens returns all currently valid access tokens (for global admin view).
func (r *OAuthRepo) ListAllActiveAccessTokens(ctx context.Context) ([]OAuthAccessToken, error) {
	query := `
		SELECT token_hash, client_id, user_id, scope, granted_mcp_key_ids, expires_at, created_at
		FROM oauth_access_tokens
		WHERE expires_at > NOW()
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []OAuthAccessToken
	for rows.Next() {
		t, err := scanOAuthAccessToken(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, nil
}

func scanOAuthAccessToken(rows pgx.Rows) (OAuthAccessToken, error) {
	var t OAuthAccessToken
	var idsJSON []byte
	if err := rows.Scan(&t.TokenHash, &t.ClientID, &t.UserID, &t.Scope, &idsJSON, &t.ExpiresAt, &t.CreatedAt); err != nil {
		return OAuthAccessToken{}, err
	}
	if err := decodeGrantedMCPKeyIDs(idsJSON, &t.GrantedMCPKeyIDs); err != nil {
		return OAuthAccessToken{}, fmt.Errorf("decode granted_mcp_key_ids: %w", err)
	}
	return t, nil
}

// RevokeAllAccessTokensForClient deletes all active tokens for a client (bulk revoke).
func (r *OAuthRepo) RevokeAllAccessTokensForClient(ctx context.Context, clientID string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM oauth_access_tokens 
		WHERE client_id = $1 AND expires_at > NOW()
	`, clientID)
	if err != nil {
		return err
	}
	return r.RevokeRefreshTokensForClient(ctx, clientID)
}

// CreateRefreshToken stores a new refresh token hash.
func (r *OAuthRepo) CreateRefreshToken(ctx context.Context, token *OAuthRefreshToken) error {
	idsJSON, err := encodeGrantedMCPKeyIDs(token.GrantedMCPKeyIDs)
	if err != nil {
		return fmt.Errorf("marshal granted_mcp_key_ids: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO oauth_refresh_tokens
		(token_hash, client_id, user_id, scope, granted_mcp_key_ids, expires_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6)
	`, token.TokenHash, token.ClientID, token.UserID, token.Scope, string(idsJSON), token.ExpiresAt)
	return err
}

// ConsumeRefreshToken validates, revokes, and returns a refresh token (one-time use).
func (r *OAuthRepo) ConsumeRefreshToken(ctx context.Context, tokenHash, clientID string) (*OAuthRefreshToken, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := `
		SELECT token_hash, client_id, user_id, scope, granted_mcp_key_ids, expires_at, revoked_at, created_at
		FROM oauth_refresh_tokens
		WHERE token_hash = $1 AND client_id = $2
		FOR UPDATE
	`
	var t OAuthRefreshToken
	var idsJSON []byte
	var revokedAt *time.Time
	err = tx.QueryRow(ctx, query, tokenHash, clientID).Scan(
		&t.TokenHash, &t.ClientID, &t.UserID, &t.Scope, &idsJSON,
		&t.ExpiresAt, &revokedAt, &t.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if revokedAt != nil || time.Now().After(t.ExpiresAt) {
		return nil, nil
	}
	if err := decodeGrantedMCPKeyIDs(idsJSON, &t.GrantedMCPKeyIDs); err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `UPDATE oauth_refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &t, nil
}

// RevokeRefreshTokensForClient soft-revokes all active refresh tokens for a client.
func (r *OAuthRepo) RevokeRefreshTokensForClient(ctx context.Context, clientID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE oauth_refresh_tokens
		SET revoked_at = NOW()
		WHERE client_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`, clientID)
	return err
}

// CreateConsent records that a user granted consent to a client.
func (r *OAuthRepo) CreateConsent(ctx context.Context, consent *OAuthConsent) error {
	idsJSON, err := encodeGrantedMCPKeyIDs(consent.GrantedMCPKeyIDs)
	if err != nil {
		return fmt.Errorf("marshal granted_mcp_key_ids: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO oauth_consents (user_id, client_id, scope, granted_mcp_key_ids)
		VALUES ($1, $2, $3, $4::jsonb)
	`, consent.UserID, consent.ClientID, consent.Scope, string(idsJSON))
	return err
}

// RevokeConsent soft-revokes a consent record.
func (r *OAuthRepo) RevokeConsent(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE oauth_consents SET revoked_at = NOW() WHERE id = $1`, id)
	return err
}
