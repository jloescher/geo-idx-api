package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/quantyralabs/idx-api/internal/mcp/auth"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// Tier classifies tool cost for rate limiting.
type Tier int

const (
	TierCheap Tier = iota
	TierMedium
	TierExpensive
)

var tierLimits = map[Tier]int{
	TierCheap:     600,
	TierMedium:    120,
	TierExpensive: 30,
}

var tierWindows = map[Tier]time.Duration{
	TierCheap:     time.Minute,
	TierMedium:    time.Minute,
	TierExpensive: time.Minute,
}

// Limiter enforces per-key rolling-window MCP tool usage limits.
type Limiter struct {
	repo *repository.MCPUsageRepo
}

func NewLimiter(repo *repository.MCPUsageRepo) *Limiter {
	return &Limiter{repo: repo}
}

// Allow records usage when under the tier limit.
func (l *Limiter) Allow(ctx context.Context, session auth.AuthSession, toolName string, tier Tier) error {
	if l == nil || l.repo == nil {
		return nil
	}
	limit, ok := tierLimits[tier]
	if !ok {
		limit = tierLimits[TierMedium]
	}
	window := tierWindows[tier]

	var keyID *int64
	clientID := ""
	if session.MCPKey != nil {
		id := session.MCPKey.ID
		keyID = &id
	}
	if session.OAuthToken != nil {
		clientID = session.OAuthToken.ClientID
	}

	since := time.Now().Add(-window)
	count, err := l.repo.CountSince(ctx, keyID, clientID, toolName, since)
	if err != nil {
		return nil // fail open on read errors
	}
	if count >= limit {
		return fmt.Errorf("rate limit exceeded for tool %s (%d/min)", toolName, limit)
	}
	_ = l.repo.Record(ctx, keyID, clientID, toolName)
	return nil
}
