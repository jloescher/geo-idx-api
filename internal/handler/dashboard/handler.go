package dashboard

import (
	"errors"
	"log/slog"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/quantyralabs/idx-api/internal/auth/password"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/auth"
	dashsvc "github.com/quantyralabs/idx-api/internal/service/dashboard"
	"github.com/quantyralabs/idx-api/internal/service/dns"
	"github.com/quantyralabs/idx-api/internal/service/mls"
	"github.com/quantyralabs/idx-api/internal/web"
)

// Handler provides invite-only dashboard (domains, API keys, monitoring).
type Handler struct {
	cfg            config.Config
	db             *repository.DB
	tokens         *repository.TokenRepo
	monitoringRepo *repository.MonitoringRepo
	provision      *dashsvc.ProvisionService
	monitoring     *dashsvc.MonitoringService
	feeds          *mls.Resolver
	invitations    *auth.InvitationService
	sessions       *session.Store
	logger         *slog.Logger
}

// NewHandler constructs the dashboard handler.
func NewHandler(cfg config.Config, db *repository.DB, logger *slog.Logger) *Handler {
	store := session.New(session.Config{
		Expiration:     cfg.Auth.SessionLifetime,
		Storage:        newPGSessionStorage(db.Pool),
		KeyLookup:      "cookie:session_id",
		CookiePath:     "/",
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
	})
	tokens := repository.NewTokenRepo(db)
	return &Handler{
		cfg:            cfg,
		db:             db,
		tokens:         tokens,
		monitoringRepo: repository.NewMonitoringRepo(db),
		provision:      dashsvc.NewProvisionService(db, tokens),
		monitoring:     dashsvc.NewMonitoringService(cfg, db),
		feeds:          mls.NewResolver(cfg),
		invitations:    auth.NewInvitationService(cfg, db),
		sessions:       store,
		logger:         logger,
	}
}

func (h *Handler) Register(app *fiber.App) {
	app.Get("/login", h.LoginForm)
	app.Post("/login", h.Login)
	app.Get("/logout", h.Logout)

	app.Get("/dashboard", h.requireAuth, h.DashboardHome)
	app.Get("/dashboard/monitoring", h.requireAuth, h.MonitoringPage)
	app.Get("/dashboard/monitoring/data", h.SessionAuthMiddleware, h.MonitoringJSON)
	app.Get("/dashboard/domains", h.requireAuth, h.DomainsPage)
	app.Get("/dashboard/setup", h.requireAuth, h.redirectToDomains)
	app.Get("/dashboard/api-keys", h.requireAuth, h.redirectToDomains)
	app.Get("/dashboard/invite", h.requireAuth, h.requireAdmin, h.InvitePage)

	app.Post("/dashboard/domains", h.requireAuth, h.StoreDomain)
	app.Post("/dashboard/domains/:id/verify-txt", h.requireAuth, h.VerifyTXT)
	app.Post("/dashboard/domains/:id/regenerate-token", h.requireAuth, h.RegenerateDomainToken)
	app.Post("/dashboard/domains/:id/delete", h.requireAuth, h.DeleteDomainBundle)
	app.Post("/dashboard/api-tokens", h.requireAuth, h.deprecateManualToken)
	app.Post("/dashboard/api-tokens/staging", h.requireAuth, h.deprecateManualToken)
	app.Delete("/dashboard/api-tokens/:id", h.requireAuth, h.RevokeToken)
	app.Post("/dashboard/api-tokens/:id/revoke", h.requireAuth, h.RevokeToken)
	app.Post("/dashboard/invitations", h.requireAuth, h.requireAdmin, h.CreateInvitation)
	app.Get("/invite/:token", h.InviteRegisterForm)
	app.Post("/invite/:token", h.AcceptInvitation)
}

func (h *Handler) requireAuth(c *fiber.Ctx) error {
	uid, _, ok := bindSessionUser(c, h.sessions)
	if !ok {
		return c.Redirect("/login")
	}
	c.Locals("user_id", uid)
	return c.Next()
}

// RequireAdmin restricts routes to dashboard users with is_admin.
func (h *Handler) RequireAdmin(c *fiber.Ctx) error {
	return h.requireAdmin(c)
}

func (h *Handler) requireAdmin(c *fiber.Ctx) error {
	uid, _ := c.Locals("user_id").(int64)
	pool, err := h.db.ReadPool(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	var isAdmin bool
	err = pool.QueryRow(c.Context(), `SELECT is_admin FROM users WHERE id = $1`, uid).Scan(&isAdmin)
	if err != nil || !isAdmin {
		return fiber.NewError(fiber.StatusForbidden, "admin only")
	}
	return c.Next()
}

func (h *Handler) LoginForm(c *fiber.Ctx) error {
	form := `<form method="post" action="/login" class="form-stack">
<label>Email <input name="email" type="email" required autocomplete="email"></label>
<label>Password <input name="password" type="password" required autocomplete="current-password"></label>
<button type="submit" class="btn btn-primary">Sign in</button>
</form>`
	return c.Type("html").SendString(web.LoginPage(form))
}

func (h *Handler) Login(c *fiber.Ctx) error {
	email := c.FormValue("email")
	pass := c.FormValue("password")
	pool, err := h.db.ReadPool(c.Context())
	if err != nil {
		return c.Status(500).SendString("Database error")
	}
	var id int64
	var hash string
	err = pool.QueryRow(c.Context(), `SELECT id, password FROM users WHERE LOWER(email) = LOWER($1)`, email).Scan(&id, &hash)
	if err != nil || password.Verify(pass, hash) != nil {
		return c.Status(401).SendString("Invalid credentials")
	}
	if password.NeedsRehash(hash) {
		if upgraded, err := password.Hash(pass, password.DefaultParams); err == nil {
			_, _ = h.db.Pool.Exec(c.Context(), `UPDATE users SET password = $1, updated_at = NOW() WHERE id = $2`, upgraded, id)
		}
	}
	sess, _ := h.sessions.Get(c)
	sess.Set("user_id", id)
	if err := sess.Save(); err != nil {
		h.logger.Error("dashboard session save failed", "error", err, "user_id", id)
		return c.Status(500).SendString("Could not start session — try again.")
	}
	return c.Redirect("/dashboard/monitoring")
}

func (h *Handler) Logout(c *fiber.Ctx) error {
	sess, _ := h.sessions.Get(c)
	_ = sess.Destroy()
	return c.Redirect("/login")
}

func (h *Handler) DashboardHome(c *fiber.Ctx) error {
	return c.Redirect("/dashboard/monitoring")
}

func (h *Handler) MonitoringPage(c *fiber.Ctx) error {
	uid, _ := c.Locals("user_id").(int64)
	data, err := h.loadPageData(c, uid, "", "")
	if err != nil {
		return c.Status(500).SendString("Database error")
	}
	snap, err := h.monitoring.BuildSnapshot(c.Context())
	if err != nil {
		h.logger.Warn("monitoring bootstrap", "error", err)
	} else {
		data.MonitoringBootstrap = snap
	}
	return c.Type("html").SendString(renderMonitoringPage(data))
}

func (h *Handler) redirectToDomains(c *fiber.Ctx) error {
	target := "/dashboard/domains"
	if q := c.Context().URI().QueryString(); len(q) > 0 {
		target += "?" + string(q)
	}
	return c.Redirect(target)
}

func (h *Handler) DomainsPage(c *fiber.Ctx) error {
	uid, _ := c.Locals("user_id").(int64)
	data, err := h.loadPageData(c, uid, "", "")
	if err != nil {
		return c.Status(500).SendString("Database error")
	}
	data.ProvisionFlash = h.takeProvisionFlash(c)
	data.TokenReveals = h.takeTokenReveals(c)
	if c.Query("verified") == "1" {
		data.VerifySuccess = "Domain verified successfully."
	}
	if c.Query("deleted") == "1" {
		data.VerifySuccess = "Domain bundle deleted."
	}
	if msg := c.Query("verify_error"); msg != "" {
		data.VerifyError = msg
	}
	if msg := c.Query("error"); msg != "" {
		data.DomainError = msg
	}
	return c.Type("html").SendString(renderDomainsPage(data))
}

func (h *Handler) InvitePage(c *fiber.Ctx) error {
	uid, _ := c.Locals("user_id").(int64)
	data, err := h.loadPageData(c, uid, "", "")
	if err != nil {
		return c.Status(500).SendString("Database error")
	}
	return c.Type("html").SendString(renderInvitePage(data))
}

func (h *Handler) loadPageData(c *fiber.Ctx, uid int64, domainErr, submittedHost string) (PageData, error) {
	pool, err := h.db.ReadPool(c.Context())
	if err != nil {
		return PageData{}, err
	}
	data := PageData{
		Feeds:         h.feeds.Catalog(),
		DomainError:   domainErr,
		SubmittedHost: submittedHost,
	}
	setup, err := h.monitoringRepo.UserSetup(c.Context(), uid)
	if err != nil {
		return PageData{}, err
	}
	data.Setup = setup

	rows, err := pool.Query(c.Context(), `
		SELECT id, domain_slug, verification_status,
		       COALESCE(txt_verification_name, ''), COALESCE(txt_verification_value, ''),
		       parent_domain_id, is_staging
		FROM domains WHERE user_id = $1 ORDER BY id
	`, uid)
	if err != nil {
		return PageData{}, err
	}
	defer rows.Close()

	prodRoots := []DomainRow{}
	stagingByParent := map[int64]DomainRow{}
	for rows.Next() {
		var d DomainRow
		var parentID *int64
		if err := rows.Scan(&d.ID, &d.Slug, &d.Status, &d.TXTName, &d.TXTValue, &parentID, &d.IsStaging); err != nil {
			return PageData{}, err
		}
		tok, err := h.tokens.FindByDomain(c.Context(), uid, d.ID)
		if err != nil {
			return PageData{}, err
		}
		d.Token = tokMetaFromRepo(tok)
		if parentID != nil {
			d.ParentID = *parentID
			stagingByParent[*parentID] = d
			continue
		}
		if d.IsStaging {
			continue
		}
		prodRoots = append(prodRoots, d)
	}
	if err := rows.Err(); err != nil {
		return PageData{}, err
	}

	for _, d := range prodRoots {
		bundle := DomainBundle{Production: d}
		if st, ok := stagingByParent[d.ID]; ok {
			stCopy := st
			bundle.Staging = &stCopy
		}
		data.Bundles = append(data.Bundles, bundle)
	}

	_ = pool.QueryRow(c.Context(), `SELECT is_admin FROM users WHERE id = $1`, uid).Scan(&data.IsAdmin)
	return data, nil
}

func tokMetaFromRepo(t *repository.TokenMeta) *TokenMeta {
	if t == nil {
		return nil
	}
	return &TokenMeta{
		ID:        t.ID,
		Name:      t.Name,
		LastUsed:  t.LastUsed,
		NeverUsed: t.NeverUsed,
	}
}

func (h *Handler) StoreDomain(c *fiber.Ctx) error {
	uid, _ := c.Locals("user_id").(int64)
	slug := strings.ToLower(strings.TrimSpace(c.FormValue("domain_slug")))
	datasets := formValues(c, "mls_datasets[]")
	if len(datasets) == 0 {
		data, err := h.loadPageData(c, uid, "Select at least one MLS dataset.", slug)
		if err != nil {
			return c.Status(500).SendString("Database error")
		}
		return c.Type("html").SendString(renderDomainsPage(data))
	}
	result, err := h.provision.ProvisionBundle(c.Context(), uid, dashsvc.ProvisionRequest{
		Hostname: slug,
		Datasets: datasets,
	})
	if err != nil {
		msg := err.Error()
		switch {
		case errors.Is(err, dashsvc.ErrEmptyHostname):
			msg = "Hostname is required."
		case errors.Is(err, dashsvc.ErrInvalidHostname):
			msg = "Hostname cannot start with staging."
		case errors.Is(err, dashsvc.ErrNoDatasets):
			msg = "Select at least one MLS dataset."
		case errors.Is(err, dashsvc.ErrDuplicateDomain):
			msg = "That domain is already registered."
		}
		data, loadErr := h.loadPageData(c, uid, msg, slug)
		if loadErr != nil {
			return c.Status(500).SendString("Database error")
		}
		return c.Type("html").SendString(renderDomainsPage(data))
	}
	if err := h.setProvisionFlash(c, result); err != nil {
		h.logger.Warn("provision flash", "error", err)
	}
	return c.Redirect("/dashboard/domains#verify")
}

func (h *Handler) VerifyTXT(c *fiber.Ctx) error {
	uid, _ := c.Locals("user_id").(int64)
	id := c.Params("id")
	var txtHost, txtVal, slug string
	readPool, err := h.db.ReadPool(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	err = readPool.QueryRow(c.Context(), `
		SELECT txt_verification_name, txt_verification_value, domain_slug FROM domains WHERE id = $1 AND user_id = $2
	`, id, uid).Scan(&txtHost, &txtVal, &slug)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "domain not found")
	}
	ok, err := dns.VerifyTXT(c.Context(), txtHost, txtVal)
	if err != nil {
		return c.Redirect("/dashboard/domains?verify_error=" + urlQuery("DNS lookup failed. Try again in a moment."))
	}
	if !ok {
		_, _ = h.db.Pool.Exec(c.Context(), `
			UPDATE domains SET verification_checked_at = NOW(), updated_at = NOW() WHERE id = $1
		`, id)
		return c.Redirect("/dashboard/domains?verify_error=" + urlQuery("TXT record not found for "+slug+". Publish the record at your DNS host, then try again."))
	}
	_, err = h.db.Pool.Exec(c.Context(), `
		UPDATE domains SET verification_status = 'verified', txt_verified_at = NOW(),
			verification_checked_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, id, uid)
	if err != nil {
		return err
	}
	return c.Redirect("/dashboard/domains?verified=1#domains")
}

func (h *Handler) RegenerateDomainToken(c *fiber.Ctx) error {
	uid, _ := c.Locals("user_id").(int64)
	domainID := parseInt64(c.Params("id"))
	readPool, err := h.db.ReadPool(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	var isStaging bool
	err = readPool.QueryRow(c.Context(), `
		SELECT is_staging FROM domains WHERE id = $1 AND user_id = $2
	`, domainID, uid).Scan(&isStaging)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "domain not found")
	}
	name := "Production"
	if isStaging {
		name = "Staging"
	}
	plain, err := h.tokens.RegenerateForDomain(c.Context(), uid, domainID, name, []string{"idx:full"})
	if err != nil {
		h.logger.Warn("regenerate token", "error", err, "domain_id", domainID)
		return c.Redirect("/dashboard/domains?error=" + urlQuery("Could not regenerate API key."))
	}
	if err := h.setTokenReveal(c, domainID, plain); err != nil {
		h.logger.Warn("token reveal flash", "error", err)
	}
	return c.Redirect("/dashboard/domains#domain-" + strconv.FormatInt(domainID, 10))
}

func (h *Handler) DeleteDomainBundle(c *fiber.Ctx) error {
	uid, _ := c.Locals("user_id").(int64)
	domainID := parseInt64(c.Params("id"))
	if err := repository.DeleteDomainBundle(c.Context(), h.db, uid, domainID); err != nil {
		msg := "Could not delete domain."
		if strings.Contains(err.Error(), "not staging") || strings.Contains(err.Error(), "production") {
			msg = "Delete the production hostname row, not staging."
		}
		return c.Redirect("/dashboard/domains?error=" + urlQuery(msg))
	}
	return c.Redirect("/dashboard/domains?deleted=1#domains")
}

func (h *Handler) deprecateManualToken(c *fiber.Ctx) error {
	return c.Redirect("/dashboard/domains?error=" + urlQuery("API keys are managed per domain. Use Regenerate on the Domains page."))
}

func (h *Handler) RevokeToken(c *fiber.Ctx) error {
	uid, _ := c.Locals("user_id").(int64)
	_ = h.tokens.Revoke(c.Context(), uid, parseInt64(c.Params("id")))
	return c.Redirect("/dashboard/domains")
}

func (h *Handler) CreateInvitation(c *fiber.Ctx) error {
	uid, _ := c.Locals("user_id").(int64)
	plain, err := h.invitations.Create(c.Context(), uid, c.FormValue("email"))
	if err != nil {
		return c.Status(400).SendString(err.Error())
	}
	link := "/invite/" + plain
	body := `<div class="card"><h1>Invitation created</h1><p>Share this link (shown once):</p><div class="token-box">` + web.Esc(link) + `</div><p><a class="btn btn-primary" href="/dashboard/invite">Back</a></p></div>`
	return c.Type("html").SendString(web.Page("Invitation", body))
}

func (h *Handler) InviteRegisterForm(c *fiber.Ctx) error {
	form := `<form method="post" class="form-stack">
<label>Name <input name="name" type="text" required></label>
<label>Password <input name="password" type="password" required minlength="8"></label>
<button type="submit" class="btn btn-primary">Create account</button>
</form>`
	return c.Type("html").SendString(web.LoginPage(form))
}

func (h *Handler) AcceptInvitation(c *fiber.Ctx) error {
	err := h.invitations.Accept(c.Context(), c.Params("token"), c.FormValue("name"), c.FormValue("password"))
	if err != nil {
		return c.Status(400).SendString(err.Error())
	}
	return c.Redirect("/login")
}

func formValues(c *fiber.Ctx, key string) []string {
	var out []string
	c.Request().PostArgs().VisitAll(func(k, v []byte) {
		if string(k) == key {
			out = append(out, string(v))
		}
	})
	return out
}

func parseInt64(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func urlQuery(s string) string {
	return url.QueryEscape(s)
}
