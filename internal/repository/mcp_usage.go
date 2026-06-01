package repository

import (
	"context"
	"fmt"
	"time"
)

// MCPUsageRepo tracks MCP tool usage for rate limiting.
type MCPUsageRepo struct {
	db *DB
}

func NewMCPUsageRepo(db *DB) *MCPUsageRepo {
	return &MCPUsageRepo{db: db}
}

func (r *MCPUsageRepo) CountSince(ctx context.Context, keyID *int64, oauthClientID, toolName string, since time.Time) (int, error) {
	pool := r.db.Pool
	if pool == nil {
		return 0, fmt.Errorf("primary database pool not available")
	}
	var count int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM mcp_tool_usage
		WHERE tool_name = $1
		  AND created_at >= $2
		  AND (
		    ($3::bigint IS NOT NULL AND mcp_key_id = $3)
		    OR ($4 <> '' AND oauth_client_id = $4)
		  )
	`, toolName, since, keyID, oauthClientID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count mcp_tool_usage: %w", err)
	}
	return count, nil
}

func (r *MCPUsageRepo) Record(ctx context.Context, keyID *int64, oauthClientID, toolName string) error {
	pool := r.db.Pool
	if pool == nil {
		return fmt.Errorf("primary database pool not available")
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO mcp_tool_usage (mcp_key_id, oauth_client_id, tool_name)
		VALUES ($1, NULLIF($2, ''), $3)
	`, keyID, oauthClientID, toolName)
	if err != nil {
		return fmt.Errorf("insert mcp_tool_usage: %w", err)
	}
	return nil
}
