package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// Handler handles OAuth 2.1 endpoints for Custom MCP connectors (e.g. Grok Web).
type Handler struct {
	cfg        config.Config
	db         *repository.DB
	logger     *slog.Logger
	clientRepo *repository.OAuthClientRepo
	oauthRepo  *repository.OAuthRepo
	mcpKeyRepo *repository.MCPKeyRepo
}

// NewHandler creates a new OAuth handler.
func NewHandler(cfg config.Config, db *repository.DB, mcpKeyRepo *repository.MCPKeyRepo, logger *slog.Logger) *Handler {
	clientRepo := repository.NewOAuthClientRepo(db)
	oauthRepo := repository.NewOAuthRepo(db)

	return &Handler{
		cfg:        cfg,
		db:         db,
		logger:     logger,
		clientRepo: clientRepo,
		oauthRepo:  oauthRepo,
		mcpKeyRepo: mcpKeyRepo,
	}
}

// RegisterRoutes mounts the OAuth routes.
func (h *Handler) RegisterRoutes(app fiber.Router) {
	app.Get("/oauth/authorize", h.Authorize)
	app.Post("/oauth/authorize", h.Consent)
	app.Post("/oauth/token", h.Token)
}

// Authorize handles GET /oauth/authorize
// For a functional version: requires dashboard session, shows minimal consent.
func (h *Handler) Authorize(c *fiber.Ctx) error {
	clientID := c.Query("client_id")
	redirectURI := c.Query("redirect_uri")
	responseType := c.Query("response_type", "code")
	codeChallenge := c.Query("code_challenge")
	codeChallengeMethod := c.Query("code_challenge_method")
	scope := c.Query("scope", defaultScopeString)
	state := c.Query("state")

	if clientID == "" || redirectURI == "" || responseType != "code" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error":             "invalid_request",
			"error_description": "client_id, redirect_uri and response_type=code are required",
		})
	}

	// Validate that the client is registered (respects admin-only client creation)
	client, err := h.clientRepo.FindByClientID(c.Context(), clientID)
	if err != nil || client == nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error":             "invalid_client",
			"error_description": "Unknown or unregistered client_id. An admin must create it first via the dashboard.",
		})
	}

	if !redirectURIAllowedForClient(clientID, redirectURI, client.RedirectURIs) {
		h.logger.Warn("oauth authorize rejected redirect_uri",
			"client_id", clientID,
			"redirect_uri", redirectURI,
			"normalized", normalizeRedirectURI(redirectURI),
		)
		return c.Status(http.StatusBadRequest).JSON(redirectURIRejectedResponse(clientID, redirectURI))
	}

	if err := validatePKCEForAuthorize(codeChallenge, codeChallengeMethod); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error":             "invalid_request",
			"error_description": err.Error(),
		})
	}

	validatedScopes, err := ParseAndValidateScopes(scope)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error":             "invalid_scope",
			"error_description": err.Error(),
		})
	}
	scope = ScopeString(validatedScopes)

	// Check if user is logged in via dashboard session
	if uid, ok := c.Locals("user_id").(int64); !ok || uid == 0 {
		// Not logged in → redirect to dashboard login with return url
		returnURI := c.OriginalURL()
		loginURL := fmt.Sprintf("/login?return_to=%s", url.QueryEscape(returnURI))
		return c.Redirect(loginURL)
	}

	html := renderConsentPage(client.Name, clientID, redirectURI, codeChallenge, codeChallengeMethod, scope, state)
	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

// Consent handles POST /oauth/authorize (user consented)
func (h *Handler) Consent(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(int64)
	if !ok || userID == 0 {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "login_required"})
	}

	clientID := c.FormValue("client_id")
	redirectURI := c.FormValue("redirect_uri")
	codeChallenge := c.FormValue("code_challenge")
	codeChallengeMethod := c.FormValue("code_challenge_method")
	scope := c.FormValue("scope", defaultScopeString)
	state := c.FormValue("state")
	consent := c.FormValue("consent")

	client, err := h.clientRepo.FindByClientID(c.Context(), clientID)
	if err != nil || client == nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error":             "invalid_client",
			"error_description": "Unknown or unregistered client_id",
		})
	}
	if !redirectURIAllowedForClient(clientID, redirectURI, client.RedirectURIs) {
		h.logger.Warn("oauth consent rejected redirect_uri",
			"client_id", clientID,
			"redirect_uri", redirectURI,
			"normalized", normalizeRedirectURI(redirectURI),
		)
		return c.Status(http.StatusBadRequest).JSON(redirectURIRejectedResponse(clientID, redirectURI))
	}

	if consent != "granted" {
		u, _ := url.Parse(redirectURI)
		q := u.Query()
		q.Set("error", "access_denied")
		if state != "" {
			q.Set("state", state)
		}
		u.RawQuery = q.Encode()
		return c.Redirect(u.String(), fiber.StatusSeeOther)
	}

	validatedScopes, err := ParseAndValidateScopes(scope)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error":             "invalid_scope",
			"error_description": err.Error(),
		})
	}
	scope = ScopeString(validatedScopes)

	if strings.TrimSpace(codeChallengeMethod) == "" {
		codeChallengeMethod = "S256"
	}

	// Generate authorization code
	code := generateRandomString(32)

	authCode := &repository.OAuthAuthorizationCode{
		Code:                code,
		ClientID:            clientID,
		UserID:              userID,
		RedirectURI:         redirectURI,
		Scope:               scope,
		CodeChallenge:       &codeChallenge,
		CodeChallengeMethod: &codeChallengeMethod,
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	}

	if err := h.oauthRepo.CreateAuthorizationCode(c.Context(), authCode); err != nil {
		h.logger.Error("failed to store auth code", "error", err)
		return c.Status(http.StatusInternalServerError).SendString("Internal error")
	}

	redirectURL, err := buildAuthorizationRedirectURL(redirectURI, code, state)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error":             "invalid_request",
			"error_description": "invalid redirect_uri",
		})
	}

	return c.Redirect(redirectURL, fiber.StatusSeeOther)
}

// CreateClient is an admin-only endpoint to register new OAuth clients.
// This respects the decision to only allow client creation via admin action (no auto-seeding or hardcoded clients).
func (h *Handler) CreateClient(c *fiber.Ctx) error {
	type req struct {
		Name         string   `json:"name"`
		ClientID     string   `json:"client_id"`
		RedirectURIs []string `json:"redirect_uris"`
		IsTrusted    bool     `json:"is_trusted"`
	}

	var body req
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json")
	}

	if body.Name == "" || body.ClientID == "" || len(body.RedirectURIs) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "name, client_id and at least one redirect_uri are required")
	}

	// Very simple insert for functional v1.
	query := `
		INSERT INTO oauth_clients (name, client_id, redirect_uris, is_trusted, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (client_id) DO NOTHING
		RETURNING id
	`

	var id int64
	err := h.db.Pool.QueryRow(c.Context(), query,
		body.Name, body.ClientID, body.RedirectURIs, body.IsTrusted, c.Locals("user_id"),
	).Scan(&id)

	if err != nil {
		return fiber.NewError(fiber.StatusConflict, "client_id already exists or creation failed")
	}

	return c.JSON(fiber.Map{
		"id":        id,
		"client_id": body.ClientID,
		"message":   "Client created successfully. Use this client_id when adding the connector in Grok Web.",
	})
}

func (h *Handler) ListClients(c *fiber.Ctx) error {
	uid := c.Locals("user_id").(int64)
	clients, err := h.clientRepo.ListByCreator(c.Context(), uid)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(clients)
}

// UpdateClient replaces redirect URIs for an existing OAuth client (admin-only).
func (h *Handler) UpdateClient(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid client id")
	}

	type req struct {
		RedirectURIs []string `json:"redirect_uris"`
	}
	var body req
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json")
	}
	if len(body.RedirectURIs) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "redirect_uris is required")
	}

	uid := c.Locals("user_id").(int64)
	if err := h.clientRepo.UpdateRedirectURIs(c.Context(), int64(id), uid, body.RedirectURIs); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"ok": true, "redirect_uris": body.RedirectURIs})
}

func (h *Handler) RevokeClient(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid client id")
	}

	uid := c.Locals("user_id").(int64)
	if err := h.clientRepo.Delete(c.Context(), int64(id), uid); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *Handler) ListAccessTokens(c *fiber.Ctx) error {
	clientID := c.Query("client_id")
	if clientID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "client_id is required")
	}

	tokens, err := h.oauthRepo.ListActiveAccessTokensByClient(c.Context(), clientID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	enriched := h.enrichAccessTokensWithKeyNames(c.Context(), tokens)
	return c.JSON(enriched)
}

// enrichAccessTokensWithKeyNames resolves granted_mcp_key_ids into human-readable names for admin UIs.
func (h *Handler) enrichAccessTokensWithKeyNames(ctx context.Context, tokens []repository.OAuthAccessToken) []map[string]interface{} {
	if len(tokens) == 0 {
		return []map[string]interface{}{}
	}

	allIDs := map[int64]bool{}
	for _, t := range tokens {
		for _, id := range t.GrantedMCPKeyIDs {
			allIDs[id] = true
		}
	}

	ids := make([]int64, 0, len(allIDs))
	for id := range allIDs {
		ids = append(ids, id)
	}

	names, _ := h.mcpKeyRepo.GetNamesByIDs(ctx, ids)

	result := make([]map[string]interface{}, len(tokens))
	for i, t := range tokens {
		keyNames := []string{}
		for _, id := range t.GrantedMCPKeyIDs {
			if name, ok := names[id]; ok && name != "" {
				keyNames = append(keyNames, name)
			} else {
				keyNames = append(keyNames, fmt.Sprintf("#%d", id))
			}
		}

		result[i] = map[string]interface{}{
			"token_hash":          t.TokenHash,
			"client_id":           t.ClientID,
			"user_id":             t.UserID,
			"scope":               t.Scope,
			"granted_mcp_key_ids": t.GrantedMCPKeyIDs,
			"granted_key_names":   keyNames,
			"expires_at":          t.ExpiresAt,
			"created_at":          t.CreatedAt,
		}
	}
	return result
}

func (h *Handler) RevokeAccessToken(c *fiber.Ctx) error {
	tokenHash := c.Params("token_hash")
	if tokenHash == "" {
		return fiber.NewError(fiber.StatusBadRequest, "token_hash is required")
	}

	// For security, we could also verify that the client belongs to the current admin,
	// but for functional v1 we'll keep it simple.
	if err := h.oauthRepo.RevokeAccessToken(c.Context(), tokenHash); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *Handler) ListAllAccessTokens(c *fiber.Ctx) error {
	tokens, err := h.oauthRepo.ListAllActiveAccessTokens(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	enriched := h.enrichAccessTokensWithKeyNames(c.Context(), tokens)
	return c.JSON(enriched)
}

func (h *Handler) RevokeAllTokensForClient(c *fiber.Ctx) error {
	clientID := c.Params("id")
	if clientID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "client id is required")
	}

	if err := h.oauthRepo.RevokeAllAccessTokensForClient(c.Context(), clientID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}

// Token handles POST /oauth/token (authorization_code + refresh_token grants).
func (h *Handler) Token(c *fiber.Ctx) error {
	grantType := c.FormValue("grant_type")
	clientID := c.FormValue("client_id")

	switch grantType {
	case "authorization_code":
		return h.tokenAuthorizationCode(c, clientID)
	case "refresh_token":
		return h.tokenRefresh(c, clientID)
	default:
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "unsupported_grant_type"})
	}
}

func (h *Handler) tokenAuthorizationCode(c *fiber.Ctx, clientID string) error {
	code := c.FormValue("code")
	redirectURI := c.FormValue("redirect_uri")
	codeVerifier := c.FormValue("code_verifier")

	if code == "" || clientID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request"})
	}

	authCode, err := h.oauthRepo.ConsumeAuthorizationCode(c.Context(), code)
	if err != nil || authCode == nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid_grant"})
	}

	if err := validateTokenExchange(authCode, clientID, redirectURI, codeVerifier); err != nil {
		errName := "invalid_grant"
		if strings.Contains(err.Error(), "code_verifier required") {
			errName = "invalid_request"
		}
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error":             errName,
			"error_description": err.Error(),
		})
	}

	return h.issueTokens(c, authCode.ClientID, authCode.UserID, authCode.Scope)
}

func (h *Handler) tokenRefresh(c *fiber.Ctx, clientID string) error {
	refreshToken := c.FormValue("refresh_token")
	if refreshToken == "" || clientID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request"})
	}

	refresh, err := h.oauthRepo.ConsumeRefreshToken(c.Context(), hashToken(refreshToken), clientID)
	if err != nil || refresh == nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid_grant"})
	}

	return h.issueTokens(c, refresh.ClientID, refresh.UserID, refresh.Scope)
}

func (h *Handler) issueTokens(c *fiber.Ctx, clientID string, userID int64, scope string) error {
	validatedScopes, err := ParseAndValidateScopes(scope)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error":             "invalid_scope",
			"error_description": err.Error(),
		})
	}
	scope = ScopeString(validatedScopes)

	client, err := h.clientRepo.FindByClientID(c.Context(), clientID)
	if err != nil || client == nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid_client"})
	}

	keyID, err := h.mcpKeyRepo.FindOrCreateOAuthKey(c.Context(), userID, clientID, client.Name, validatedScopes)
	if err != nil {
		h.logger.Error("failed to provision oauth mcp key", "error", err, "client_id", clientID, "user_id", userID)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "server_error"})
	}

	accessTTL := h.cfg.OAuth.AccessTokenTTL
	if accessTTL <= 0 {
		accessTTL = 24 * time.Hour
	}
	refreshTTL := h.cfg.OAuth.RefreshTokenTTL
	if refreshTTL <= 0 {
		refreshTTL = 30 * 24 * time.Hour
	}

	accessToken := generateRandomString(48)
	refreshToken := generateRandomString(48)

	access := &repository.OAuthAccessToken{
		TokenHash:        hashToken(accessToken),
		ClientID:         clientID,
		UserID:           userID,
		Scope:            scope,
		GrantedMCPKeyIDs: []int64{keyID},
		ExpiresAt:        time.Now().Add(accessTTL),
	}
	if err := h.oauthRepo.CreateAccessToken(c.Context(), access); err != nil {
		h.logger.Error("failed to store access token", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "server_error"})
	}

	refresh := &repository.OAuthRefreshToken{
		TokenHash:        hashToken(refreshToken),
		ClientID:         clientID,
		UserID:           userID,
		Scope:            scope,
		GrantedMCPKeyIDs: []int64{keyID},
		ExpiresAt:        time.Now().Add(refreshTTL),
	}
	if err := h.oauthRepo.CreateRefreshToken(c.Context(), refresh); err != nil {
		h.logger.Error("failed to store refresh token", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "server_error"})
	}

	_ = h.oauthRepo.CreateConsent(c.Context(), &repository.OAuthConsent{
		UserID:           userID,
		ClientID:         clientID,
		Scope:            scope,
		GrantedMCPKeyIDs: []int64{keyID},
	})

	expiresIn := int(accessTTL.Seconds())
	return c.JSON(fiber.Map{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    expiresIn,
		"refresh_token": refreshToken,
		"scope":         scope,
	})
}

// --- Helpers ---

func generateRandomString(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

