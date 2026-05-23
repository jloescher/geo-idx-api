package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/quantyralabs/idx-api/internal/auth/password"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// InvitationService issues and accepts invite-only registrations.
type InvitationService struct {
	cfg  config.Config
	db   *repository.DB
	repo *repository.InvitationRepo
}

func NewInvitationService(cfg config.Config, db *repository.DB) *InvitationService {
	return &InvitationService{
		cfg:  cfg,
		repo: repository.NewInvitationRepo(db),
		db:   db,
	}
}

func (s *InvitationService) Create(ctx context.Context, inviterID int64, email string) (plainToken string, err error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return "", fmt.Errorf("email is required")
	}
	plain, hash, err := newInviteToken()
	if err != nil {
		return "", err
	}
	expires := time.Now().Add(s.cfg.Auth.InvitationTTL)
	if err := s.repo.Create(ctx, email, hash, inviterID, expires); err != nil {
		return "", err
	}
	return plain, nil
}

func (s *InvitationService) Accept(ctx context.Context, plainToken, name, plainPassword string) error {
	hash := hashInviteToken(plainToken)
	email, _, err := s.repo.FindOpenByHash(ctx, hash)
	if err != nil {
		return fmt.Errorf("invitation is invalid or expired")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = email
	}
	passHash, err := password.Hash(plainPassword, password.DefaultParams)
	if err != nil {
		return err
	}
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO users (name, email, password, email_verified_at, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW(), NOW())
	`, name, email, passHash)
	if err != nil {
		return fmt.Errorf("account already exists for this email")
	}
	_, err = tx.Exec(ctx, `
		UPDATE user_invitations SET accepted_at = NOW(), updated_at = NOW() WHERE token_hash = $1
	`, hash)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func newInviteToken() (plain, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	plain = hex.EncodeToString(b)
	return plain, hashInviteToken(plain), nil
}

func hashInviteToken(plain string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(plain)))
	return hex.EncodeToString(sum[:])
}

// Valid returns true when email has an unexpired, unused invitation matching token.
func (s *InvitationService) Valid(ctx context.Context, email, plainToken string) (bool, error) {
	hash := hashInviteToken(plainToken)
	pool, err := s.db.ReadPool(ctx)
	if err != nil {
		return false, err
	}
	var id int64
	err = pool.QueryRow(ctx, `
		SELECT id FROM user_invitations
		WHERE LOWER(email) = LOWER($1) AND token_hash = $2
		  AND accepted_at IS NULL AND expires_at > NOW()
	`, email, hash).Scan(&id)
	if err != nil {
		return false, nil
	}
	return id > 0, nil
}
