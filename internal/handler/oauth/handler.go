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
	scope := c.Query("scope", "monitor comps content")

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

	// Basic redirect_uri validation (must start with one of the registered URIs)
	allowed := false
	for _, allowedURI := range client.RedirectURIs {
		if strings.HasPrefix(redirectURI, allowedURI) {
			allowed = true
			break
		}
	}
	if !allowed {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error":             "invalid_request",
			"error_description": "redirect_uri is not allowed for this client",
		})
	}
	_ = redirectURI // used above

	// Check if user is logged in via dashboard session
	userID, ok := c.Locals("user_id").(int64)
	if !ok || userID == 0 {
		// Not logged in → redirect to dashboard login with return url
		returnURI := c.OriginalURL()
		loginURL := fmt.Sprintf("/login?return_to=%s", url.QueryEscape(returnURI))
		return c.Redirect(loginURL)
	}

	// For a functional version, show the user's existing MCP keys so they can choose which to grant.
	userKeys, _ := h.mcpKeyRepo.ListByCreator(c.Context(), userID)

	// Build checkboxes for the keys
	var keyCheckboxes string
	for _, k := range userKeys {
		if k.RevokedAt != nil {
			continue
		}
		keyCheckboxes += fmt.Sprintf(
			`<label style="display:block; margin-bottom:0.25rem;">
				<input type="checkbox" name="granted_key_ids" value="%d" checked>
				%s <small>(scopes: %s)</small>
			</label>`,
			k.ID, k.Name, strings.Join(k.Scopes, ", "),
		)
	}

	if keyCheckboxes == "" {
		keyCheckboxes = `<p><em>You don't have any active MCP keys yet. Create some in the dashboard first.</em></p>`
	}

	// Polished consent screen using consistent dashboard patterns
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Authorize Grok Web — Quantyra IDX</title>
  <link rel="stylesheet" href="/static/css/app.css">
  <style>
    body { background: #f8fafc; }
    .consent-container { max-width: 620px; margin: 4rem auto; padding: 0 1rem; }
    .consent-card { background: white; border-radius: 12px; box-shadow: 0 10px 15px -3px rgb(0 0 0 / 0.1); padding: 2rem; }
  </style>
</head>
<body>
  <div class="consent-container">
    <div class="consent-card">
      <h1 style="margin-bottom: 0.25rem;">Authorize Grok Web</h1>
      <p class="text-muted" style="margin-bottom: 1.5rem;">Grok Web is requesting access to your Quantyra IDX MCP tools.</p>

      <div class="setup-section">
        <h2 style="font-size: 1.1rem; margin-bottom: 0.5rem;">Requested Scopes</h2>
        <ul style="margin: 0 0 1rem 1rem; padding: 0;">
          <li><strong>monitor</strong> — View system monitoring, queues, GIS health, etc.</li>
          <li><strong>comps</strong> — Run comparable sales / BPO analysis</li>
          <li><strong>content</strong> — Safe content generation queries over listings and GIS</li>
        </ul>
      </div>

      <div class="setup-section">
        <h2 style="font-size: 1.1rem; margin-bottom: 0.5rem;">Select which MCP Keys to grant</h2>
        
        <form method="POST" action="/oauth/authorize">
          <input type="hidden" name="client_id" value="%s">
          <input type="hidden" name="redirect_uri" value="%s">
          <input type="hidden" name="code_challenge" value="%s">
          <input type="hidden" name="code_challenge_method" value="%s">
          <input type="hidden" name="scope" value="%s">

          %s

          <div style="margin: 1.25rem 0;">
            <label class="checkbox-label" style="font-weight: 500;">
              <input type="checkbox" name="consent" value="granted" required>
              <span>I authorize Grok Web to use the selected MCP keys with the scopes above.</span>
            </label>
          </div>

          <div style="display: flex; gap: 0.75rem; margin-top: 1rem;">
            <button type="submit" class="btn btn-primary">Authorize</button>
            <a href="%s" class="btn btn-secondary">Cancel</a>
          </div>
        </form>
      </div>
    </div>
    
    <p style="text-align: center; margin-top: 1.5rem; font-size: 0.8rem; color: #64748b;">
      You are authorizing access via the Quantyra IDX OAuth server.
    </p>
  </div>
</body>
</html>
`, clientID, redirectURI, codeChallenge, codeChallengeMethod, scope, keyCheckboxes, redirectURI)

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
	scope := c.FormValue("scope", "monitor comps content")
	consent := c.FormValue("consent")

	if consent != "granted" {
		u, _ := url.Parse(redirectURI)
		q := u.Query()
		q.Set("error", "access_denied")
		u.RawQuery = q.Encode()
		return c.Redirect(u.String())
	}

	// Parse which keys the user selected on the consent form
	var grantedMCPKeyIDs []int64
	for _, v := range c.Context().PostArgs().PeekMulti("granted_key_ids") {
		var id int64
		fmt.Sscanf(string(v), "%d", &id)
		if id > 0 {
			grantedMCPKeyIDs = append(grantedMCPKeyIDs, id)
		}
	}

	// If the user didn't select any, fall back to all their active keys (functional behavior)
	if len(grantedMCPKeyIDs) == 0 {
		userKeys, _ := h.mcpKeyRepo.ListByCreator(c.Context(), userID)
		for _, k := range userKeys {
			if k.RevokedAt == nil {
				grantedMCPKeyIDs = append(grantedMCPKeyIDs, k.ID)
			}
		}
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

	// For the functional version we store the granted key IDs by serializing them into the scope field
	// temporarily (a proper implementation would add a dedicated column or JSONB field).
	// When we exchange the code we will parse them back out.
	if len(grantedMCPKeyIDs) > 0 {
		ids := make([]string, len(grantedMCPKeyIDs))
		for i, id := range grantedMCPKeyIDs {
			ids[i] = fmt.Sprintf("%d", id)
		}
		authCode.Scope = scope + " granted_keys:" + strings.Join(ids, ",")
	}

	if err := h.oauthRepo.CreateAuthorizationCode(c.Context(), authCode); err != nil {
		h.logger.Error("failed to store auth code", "error", err)
		return c.Status(http.StatusInternalServerError).SendString("Internal error")
	}

	// Redirect back with code
	u, _ := url.Parse(redirectURI)
	q := u.Query()
	q.Set("code", code)
	q.Set("state", c.Query("state"))
	u.RawQuery = q.Encode()

	return c.Redirect(u.String())
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

// Token handles POST /oauth/token (Authorization Code + PKCE exchange)
func (h *Handler) Token(c *fiber.Ctx) error {
	grantType := c.FormValue("grant_type")
	if grantType != "authorization_code" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "unsupported_grant_type"})
	}

	code := c.FormValue("code")
	clientID := c.FormValue("client_id")
	codeVerifier := c.FormValue("code_verifier")

	if code == "" || clientID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request"})
	}

	// Consume the code (one-time use)
	authCode, err := h.oauthRepo.ConsumeAuthorizationCode(c.Context(), code)
	if err != nil || authCode == nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid_grant"})
	}

	// Basic PKCE validation (S256)
	if authCode.CodeChallenge != nil && *authCode.CodeChallenge != "" {
		if codeVerifier == "" {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request", "error_description": "code_verifier required"})
		}
		hash := sha256.Sum256([]byte(codeVerifier))
		encoded := base64.RawURLEncoding.EncodeToString(hash[:])
		if encoded != *authCode.CodeChallenge {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid_grant", "error_description": "PKCE verification failed"})
		}
	}

	// Parse granted key IDs that were selected during consent and stored on the auth code
	var grantedKeyIDs []int64
	if idx := strings.Index(authCode.Scope, "granted_keys:"); idx != -1 {
		keysPart := authCode.Scope[idx+len("granted_keys:"):]
		for _, part := range strings.Split(keysPart, ",") {
			var id int64
			fmt.Sscanf(strings.TrimSpace(part), "%d", &id)
			if id > 0 {
				grantedKeyIDs = append(grantedKeyIDs, id)
			}
		}
		// Clean the scope for the final token
		authCode.Scope = strings.TrimSpace(authCode.Scope[:idx])
	}

	// Fallback: if nothing was recorded, grant all user's active keys
	if len(grantedKeyIDs) == 0 {
		userKeys, _ := h.mcpKeyRepo.ListByCreator(c.Context(), authCode.UserID)
		for _, k := range userKeys {
			if k.RevokedAt == nil {
				grantedKeyIDs = append(grantedKeyIDs, k.ID)
			}
		}
	}

	accessToken := generateRandomString(48)
	tokenHash := hashToken(accessToken)

	access := &repository.OAuthAccessToken{
		TokenHash:        tokenHash,
		ClientID:         clientID,
		UserID:           authCode.UserID,
		Scope:            authCode.Scope,
		GrantedMCPKeyIDs: grantedKeyIDs,
		ExpiresAt:        time.Now().Add(1 * time.Hour),
	}

	if err := h.oauthRepo.CreateAccessToken(c.Context(), access); err != nil {
		h.logger.Error("failed to store access token", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "server_error"})
	}

	return c.JSON(fiber.Map{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   3600,
		"scope":        authCode.Scope,
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

// Note: In a real implementation we would also validate that the client_id + redirect_uri match
// a registered client, and we would store which specific mcp_key_ids were granted during consent.
