package middleware

import (
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
)

// UploadCORS allows credentialed dashboard uploads from the platform origin to a
// Cloudflare-bypass upload host (e.g. upload.idx.quantyralabs.cc).
func UploadCORS(cfg config.Config) fiber.Handler {
	allowed := uploadCORSOrigins(cfg)
	return func(c *fiber.Ctx) error {
		origin := strings.TrimSpace(c.Get("Origin"))
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				c.Set("Access-Control-Allow-Origin", origin)
				c.Set("Access-Control-Allow-Credentials", "true")
				c.Append("Vary", "Origin")
			}
		}
		if c.Method() == fiber.MethodOptions {
			if origin != "" {
				if _, ok := allowed[origin]; ok {
					c.Set("Access-Control-Allow-Methods", "POST, OPTIONS")
					c.Set("Access-Control-Allow-Headers", "Content-Type")
					c.Set("Access-Control-Max-Age", "86400")
				}
			}
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.Next()
	}
}

func uploadCORSOrigins(cfg config.Config) map[string]struct{} {
	out := make(map[string]struct{})
	for _, raw := range []string{cfg.Idx.PlatformURL, cfg.Idx.APIPublic} {
		if o := originFromPublicURL(raw); o != "" {
			out[o] = struct{}{}
		}
	}
	return out
}

func originFromPublicURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
