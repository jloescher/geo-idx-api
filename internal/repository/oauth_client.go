package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OAuthClient represents a registered OAuth client (e.g. Grok Web).
type OAuthClient struct {
	ID                 int64     `db:"id"`
	ClientID           string    `db:"client_id"`
	ClientSecretHash   *string   `db:"client_secret_hash"`
	Name               string    `db:"name"`
	RedirectURIs       []string  `db:"redirect_uris"`
	CreatedByUserID    *int64    `db:"created_by_user_id"`
	IsTrusted          bool      `db:"is_trusted"`
	CreatedAt          time.Time `db:"created_at"`
	UpdatedAt          time.Time `db:"updated_at"`
}

// OAuthClientRepo handles persistence for OAuth clients.
type OAuthClientRepo struct {
	pool *pgxpool.Pool
}

// NewOAuthClientRepo creates a new repo.
func NewOAuthClientRepo(db *DB) *OAuthClientRepo {
	return &OAuthClientRepo{pool: db.Pool}
}

// FindByClientID retrieves a client by its public client_id.
func (r *OAuthClientRepo) FindByClientID(ctx context.Context, clientID string) (*OAuthClient, error) {
	query := `
		SELECT id, client_id, client_secret_hash, name, redirect_uris, 
		       created_by_user_id, is_trusted, created_at, updated_at
		FROM oauth_clients 
		WHERE client_id = $1
		LIMIT 1
	`

	var client OAuthClient
	err := r.pool.QueryRow(ctx, query, clientID).Scan(
		&client.ID,
		&client.ClientID,
		&client.ClientSecretHash,
		&client.Name,
		&client.RedirectURIs,
		&client.CreatedByUserID,
		&client.IsTrusted,
		&client.CreatedAt,
		&client.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &client, nil
}

// Create inserts a new OAuth client.
func (r *OAuthClientRepo) Create(ctx context.Context, client *OAuthClient) (int64, error) {
	query := `
		INSERT INTO oauth_clients (client_id, name, redirect_uris, is_trusted, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	var id int64
	err := r.pool.QueryRow(ctx, query,
		client.ClientID, client.Name, client.RedirectURIs, client.IsTrusted, client.CreatedByUserID,
	).Scan(&id)
	return id, err
}

// ListByCreator returns OAuth clients created by a specific user.
func (r *OAuthClientRepo) ListByCreator(ctx context.Context, userID int64) ([]OAuthClient, error) {
	query := `
		SELECT id, client_id, name, redirect_uris, is_trusted, created_at
		FROM oauth_clients
		WHERE created_by_user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []OAuthClient
	for rows.Next() {
		var c OAuthClient
		err := rows.Scan(&c.ID, &c.ClientID, &c.Name, &c.RedirectURIs, &c.IsTrusted, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		clients = append(clients, c)
	}
	return clients, nil
}

// UpdateRedirectURIs replaces redirect URIs for a client owned by userID.
func (r *OAuthClientRepo) UpdateRedirectURIs(ctx context.Context, id int64, userID int64, redirectURIs []string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE oauth_clients
		SET redirect_uris = $3, updated_at = NOW()
		WHERE id = $1 AND created_by_user_id = $2
	`, id, userID, redirectURIs)
	return err
}

// Delete removes a client (and cascades to codes/tokens via FKs if set up).
func (r *OAuthClientRepo) Delete(ctx context.Context, id int64, userID int64) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM oauth_clients 
		WHERE id = $1 AND created_by_user_id = $2
	`, id, userID)
	return err
}
