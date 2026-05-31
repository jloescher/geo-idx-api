package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// MCPKey represents a long-lived key used to authenticate MCP clients.
type MCPKey struct {
	ID               int64     `db:"id"`
	Name             string    `db:"name"`
	KeyHash          string    `db:"key_hash"`
	Scopes           []string  `db:"scopes"` // populated from JSONB
	CreatedByUserID  int64     `db:"created_by_user_id"`
	CreatedAt        time.Time `db:"created_at"`
	LastUsedAt       *time.Time `db:"last_used_at"`
	RevokedAt        *time.Time `db:"revoked_at"`
	Notes            *string   `db:"notes"`
}

// MCPKeyRepo manages MCP access keys.
type MCPKeyRepo struct {
	db *DB
}

func NewMCPKeyRepo(db *DB) *MCPKeyRepo {
	return &MCPKeyRepo{db: db}
}

// GenerateMCPKey creates a new plaintext key and its hash.
// Format: mcp_<64 hex chars>
func GenerateMCPKey() (plaintext, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate random: %w", err)
	}
	plaintext = "mcp_" + hex.EncodeToString(b)

	sum := sha256.Sum256([]byte(plaintext))
	hash = hex.EncodeToString(sum[:])
	return plaintext, hash, nil
}

// HashMCPKey returns the sha256 hex hash of a plaintext key.
func HashMCPKey(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

// Create inserts a new MCP key and returns the plaintext (shown only once) + the record.
func (r *MCPKeyRepo) Create(ctx context.Context, name string, scopes []string, createdByUserID int64, notes *string) (plaintext string, key *MCPKey, err error) {
	plaintext, hash, err := GenerateMCPKey()
	if err != nil {
		return "", nil, err
	}

	pool := r.db.Pool // writes must go to primary
	if pool == nil {
		return "", nil, fmt.Errorf("primary database pool not available")
	}

	var id int64
	var createdAt time.Time

	err = pool.QueryRow(ctx, `
		INSERT INTO mcp_keys (name, key_hash, scopes, created_by_user_id, notes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`, name, hash, scopes, createdByUserID, notes).Scan(&id, &createdAt)
	if err != nil {
		return "", nil, fmt.Errorf("insert mcp_key: %w", err)
	}

	key = &MCPKey{
		ID:              id,
		Name:            name,
		KeyHash:         hash,
		Scopes:          scopes,
		CreatedByUserID: createdByUserID,
		CreatedAt:       createdAt,
		Notes:           notes,
	}
	return plaintext, key, nil
}

// FindValidByHash returns an active (non-revoked) key by its hash.
func (r *MCPKeyRepo) FindValidByHash(ctx context.Context, hash string) (*MCPKey, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}

	var k MCPKey
	var scopesJSON []byte

	err = pool.QueryRow(ctx, `
		SELECT id, name, key_hash, scopes, created_by_user_id, created_at, last_used_at, revoked_at, notes
		FROM mcp_keys
		WHERE key_hash = $1
		  AND revoked_at IS NULL
		LIMIT 1
	`, hash).Scan(
		&k.ID, &k.Name, &k.KeyHash, &scopesJSON,
		&k.CreatedByUserID, &k.CreatedAt, &k.LastUsedAt, &k.RevokedAt, &k.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find mcp_key by hash: %w", err)
	}

	// Decode scopes JSONB into []string
	if len(scopesJSON) > 0 {
		if err := json.Unmarshal(scopesJSON, &k.Scopes); err != nil {
			return nil, fmt.Errorf("unmarshal mcp_key scopes: %w", err)
		}
	}

	return &k, nil
}

// FindByID retrieves an MCP key by its ID (even if revoked, for admin/audit use).
func (r *MCPKeyRepo) FindByID(ctx context.Context, id int64) (*MCPKey, error) {
	query := `
		SELECT id, name, key_hash, scopes, created_by_user_id, created_at, last_used_at, revoked_at, notes
		FROM mcp_keys
		WHERE id = $1
		LIMIT 1
	`

	var k MCPKey
	var scopesJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&k.ID, &k.Name, &k.KeyHash, &scopesJSON,
		&k.CreatedByUserID, &k.CreatedAt, &k.LastUsedAt, &k.RevokedAt, &k.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find mcp_key by id: %w", err)
	}

	if len(scopesJSON) > 0 {
		if err := json.Unmarshal(scopesJSON, &k.Scopes); err != nil {
			return nil, fmt.Errorf("unmarshal mcp_key scopes: %w", err)
		}
	}

	return &k, nil
}

// GetNamesByIDs returns a map of MCP key ID -> name for the given IDs.
// Useful for enriching token data in admin UIs.
func (r *MCPKeyRepo) GetNamesByIDs(ctx context.Context, ids []int64) (map[int64]string, error) {
	if len(ids) == 0 {
		return map[int64]string{}, nil
	}

	// Build a safe IN clause with parameters
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, name 
		FROM mcp_keys 
		WHERE id IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]string)
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		result[id] = name
	}

	return result, nil
}

// ListByCreator returns active keys created by a given admin user.
func (r *MCPKeyRepo) ListByCreator(ctx context.Context, userID int64) ([]MCPKey, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(ctx, `
		SELECT id, name, key_hash, scopes, created_by_user_id, created_at, last_used_at, revoked_at, notes
		FROM mcp_keys
		WHERE created_by_user_id = $1
		  AND revoked_at IS NULL
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list mcp_keys: %w", err)
	}
	defer rows.Close()

	var keys []MCPKey
	for rows.Next() {
		var k MCPKey
		var scopesJSON []byte
		if err := rows.Scan(&k.ID, &k.Name, &k.KeyHash, &scopesJSON, &k.CreatedByUserID, &k.CreatedAt, &k.LastUsedAt, &k.RevokedAt, &k.Notes); err != nil {
			return nil, err
		}
		if len(scopesJSON) > 0 {
			_ = json.Unmarshal(scopesJSON, &k.Scopes)
		}
		keys = append(keys, k)
	}
	return keys, nil
}

// Revoke marks a key as revoked.
func (r *MCPKeyRepo) Revoke(ctx context.Context, id int64, userID int64) error {
	pool := r.db.Pool
	if pool == nil {
		return fmt.Errorf("primary database pool not available")
	}

	res, err := pool.Exec(ctx, `
		UPDATE mcp_keys
		SET revoked_at = NOW()
		WHERE id = $1
		  AND created_by_user_id = $2
		  AND revoked_at IS NULL
	`, id, userID)
	if err != nil {
		return fmt.Errorf("revoke mcp_key: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("key not found or already revoked or not owned by user")
	}
	return nil
}

// TouchLastUsed updates the last_used_at timestamp (best effort, non-blocking).
func (r *MCPKeyRepo) TouchLastUsed(ctx context.Context, id int64) {
	pool := r.db.Pool
	if pool == nil {
		return
	}
	_, _ = pool.Exec(context.Background(), `
		UPDATE mcp_keys SET last_used_at = NOW() WHERE id = $1
	`, id)
}

// HasScope returns true if the key has the given scope.
func (k *MCPKey) HasScope(scope string) bool {
	for _, s := range k.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}
