package mcpkey

import (
	"context"
	"fmt"

	"github.com/quantyralabs/idx-api/internal/repository"
)

// Service provides business logic for managing MCP access keys.
type Service struct {
	repo *repository.MCPKeyRepo
}

func NewService(repo *repository.MCPKeyRepo) *Service {
	return &Service{repo: repo}
}

// CreateKey creates a new MCP key for an admin user.
// It returns the plaintext key (only shown once) and the metadata record.
func (s *Service) CreateKey(ctx context.Context, name string, scopes []string, createdByUserID int64, notes *string) (plaintext string, key *repository.MCPKey, err error) {
	if name == "" {
		return "", nil, fmt.Errorf("key name is required")
	}
	if len(scopes) == 0 {
		scopes = []string{"monitor"} // sensible default
	}

	return s.repo.Create(ctx, name, scopes, createdByUserID, notes)
}

// ListKeys returns all active keys created by the given admin.
func (s *Service) ListKeys(ctx context.Context, userID int64) ([]repository.MCPKey, error) {
	return s.repo.ListByCreator(ctx, userID)
}

// RevokeKey revokes a key owned by the given user.
func (s *Service) RevokeKey(ctx context.Context, keyID int64, userID int64) error {
	return s.repo.Revoke(ctx, keyID, userID)
}
