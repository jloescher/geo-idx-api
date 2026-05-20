package middleware

import (
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/api/ctxkeys"
	dom "github.com/quantyralabs/idx-api/internal/domain"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// DomainToken authenticates MLS/GIS/image traffic (Laravel DomainOrTokenAuth parity).
func DomainToken(_ any, domains *repository.DomainRepo, tokens *repository.TokenRepo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			plain := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
			return handleToken(c, domains, tokens, plain)
		}
		return handleDomain(c, domains)
	}
}

func handleToken(c *fiber.Ctx, domains *repository.DomainRepo, tokens *repository.TokenRepo, plain string) error {
	tok, user, err := tokens.FindByPlaintext(c.Context(), plain)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if tok == nil || user == nil {
		return fiber.NewError(fiber.StatusForbidden, "Invalid API token.")
	}
	if !tokens.HasAbility(tok, "idx:access") && !tokens.HasAbility(tok, "idx:full") {
		return fiber.NewError(fiber.StatusForbidden, "Token is missing required IDX abilities.")
	}
	slug := resolveSlugForToken(c)
	if slug == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Missing domain identification. Send X-Domain-Slug (or ?domain=) matching a verified domain on your account.")
	}
	d, err := domains.FindActiveForUser(c.Context(), user.ID, slug)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if d == nil {
		return fiber.NewError(fiber.StatusForbidden, "Domain is not registered, inactive, or not owned by this token.")
	}
	if !d.IsVerified() {
		return fiber.NewError(fiber.StatusForbidden, "Domain must be TXT-verified before API token access is allowed.")
	}
	fullAccess := tokens.HasAbility(tok, "idx:full")
	setBridgeLocals(c, "token", d, &tok.Name, &user.ID, fullAccess)
	return c.Next()
}

func handleDomain(c *fiber.Ctx, domains *repository.DomainRepo) error {
	slug := resolveSlug(c)
	if slug == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "Missing domain identification.")
	}
	d, err := domains.FindActiveBySlug(c.Context(), slug)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if d == nil {
		return fiber.NewError(fiber.StatusForbidden, "Domain is not registered or inactive.")
	}
	setBridgeLocals(c, "domain", d, nil, nil, true)
	return c.Next()
}

func setBridgeLocals(c *fiber.Ctx, auth string, d *dom.Domain, tokenName *string, userID *int64, fullAccess bool) {
	c.Locals(ctxkeys.BridgeAuth, auth)
	c.Locals(ctxkeys.BridgeDomain, d)
	c.Locals(ctxkeys.BridgeDomainSlug, d.DomainSlug)
	c.Locals(ctxkeys.BridgeTokenName, tokenName)
	c.Locals(ctxkeys.BridgeUserID, userID)
	c.Locals(ctxkeys.BridgeFullAccess, fullAccess)
}

func resolveSlugForToken(c *fiber.Ctx) string {
	if h := strings.TrimSpace(c.Get("X-Domain-Slug")); h != "" {
		return h
	}
	if q := strings.TrimSpace(c.Query("domain")); q != "" {
		return q
	}
	return ""
}

func resolveSlug(c *fiber.Ctx) string {
	if s := resolveSlugForToken(c); s != "" {
		return s
	}
	ref := c.Get("Referer")
	if ref == "" {
		return ""
	}
	u, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}
