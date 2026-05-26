package repository

import (
	"context"
	"time"
)

// InvitationRepo manages user_invitations rows.
type InvitationRepo struct {
	db *DB
}

func NewInvitationRepo(db *DB) *InvitationRepo {
	return &InvitationRepo{db: db}
}

func (r *InvitationRepo) Create(ctx context.Context, email, tokenHash string, invitedBy int64, expiresAt time.Time) error {
	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO user_invitations (email, token_hash, invited_by, expires_at, created_at, updated_at)
		VALUES (LOWER($1), $2, $3, $4, NOW(), NOW())
	`, email, tokenHash, invitedBy, expiresAt)
	return err
}

func (r *InvitationRepo) FindOpenByHash(ctx context.Context, tokenHash string) (email string, expiresAt time.Time, err error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return "", time.Time{}, err
	}
	err = pool.QueryRow(ctx, `
		SELECT email, expires_at FROM user_invitations
		WHERE token_hash = $1 AND accepted_at IS NULL AND expires_at > NOW()
	`, tokenHash).Scan(&email, &expiresAt)
	return email, expiresAt, err
}

func (r *InvitationRepo) Accept(ctx context.Context, tokenHash string) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE user_invitations SET accepted_at = NOW(), updated_at = NOW()
		WHERE token_hash = $1 AND accepted_at IS NULL
	`, tokenHash)
	return err
}
