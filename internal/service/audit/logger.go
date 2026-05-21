package audit

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/api/ctxkeys"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// Logger writes mls_proxy_audit_logs entries.
type Logger struct {
	db *repository.DB
}

func NewLogger(db *repository.DB) *Logger {
	return &Logger{db: db}
}

func (l *Logger) Log(c *fiber.Ctx, requestType string, listingCount *int, cacheHit *string) {
	slug, _ := c.Locals(ctxkeys.MLSDomainSlug).(string)
	var tokenName *string
	if tn, ok := c.Locals(ctxkeys.MLSTokenName).(*string); ok {
		tokenName = tn
	}
	var userID *int64
	if uid, ok := c.Locals(ctxkeys.MLSUserID).(*int64); ok {
		userID = uid
	}
	ip := c.IP()
	_, _ = l.db.Pool.Exec(context.Background(), `
		INSERT INTO mls_proxy_audit_logs (domain_slug, token_name, request_type, listing_count, ip_address, user_id, cache_hit)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, nullStr(slug), tokenName, requestType, listingCount, ip, userID, cacheHit)
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
