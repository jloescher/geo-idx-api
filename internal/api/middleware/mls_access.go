package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/api/ctxkeys"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/mls"
)

// MLSAccess enforces per-domain feed allowlists (CheckMlsAccess parity).
func MLSAccess(cfg config.Config, _ *repository.DomainRepo) fiber.Handler {
	resolver := mls.NewResolver(cfg)
	return func(c *fiber.Ctx) error {
		if mls.BypassGIS(c.Path()) {
			return c.Next()
		}
		code, err := resolver.ResolveFeedCode(c)
		if err != nil {
			return err
		}
		c.Locals(ctxkeys.MLSFeedCode, code)
		c.Locals(ctxkeys.MLSFeedDef, resolver.FeedDefinition(code))
		return c.Next()
	}
}
