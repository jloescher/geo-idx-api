package dashboard

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/quantyralabs/idx-api/internal/auth/password"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/auth"
	"github.com/quantyralabs/idx-api/internal/service/dns"
	"github.com/quantyralabs/idx-api/internal/web"
)

// Handler provides invite-only dashboard (domains, API keys).
type Handler struct {
	cfg      config.Config
	db       *repository.DB
	tokens      *repository.TokenRepo
	domains     *repository.DomainRepo
	invitations *auth.InvitationService
	sessions    *session.Store
	logger      *slog.Logger
}

func NewHandler(cfg config.Config, db *repository.DB, logger *slog.Logger) *Handler {
	store := session.New(session.Config{
		Expiration: cfg.Auth.SessionLifetime,
	})
	return &Handler{
		cfg:      cfg,
		db:       db,
		tokens:      repository.NewTokenRepo(db),
		domains:     repository.NewDomainRepo(db),
		invitations: auth.NewInvitationService(cfg, db),
		sessions:    store,
		logger:      logger,
	}
}

func (h *Handler) Register(app *fiber.App) {
	app.Get("/login", h.LoginForm)
	app.Post("/login", h.Login)
	app.Get("/logout", h.Logout)
	app.Get("/dashboard", h.requireAuth, h.Dashboard)
	app.Post("/dashboard/domains", h.requireAuth, h.StoreDomain)
	app.Post("/dashboard/domains/:id/verify-txt", h.requireAuth, h.VerifyTXT)
	app.Post("/dashboard/api-tokens", h.requireAuth, h.CreateToken)
	app.Post("/dashboard/api-tokens/staging", h.requireAuth, h.CreateStagingToken)
	app.Delete("/dashboard/api-tokens/:id", h.requireAuth, h.RevokeToken)
	app.Post("/dashboard/api-tokens/:id/revoke", h.requireAuth, h.RevokeToken)
	app.Post("/dashboard/invitations", h.requireAuth, h.requireAdmin, h.CreateInvitation)
	app.Get("/invite/:token", h.InviteRegisterForm)
	app.Post("/invite/:token", h.AcceptInvitation)
}

func (h *Handler) requireAuth(c *fiber.Ctx) error {
	sess, _ := h.sessions.Get(c)
	uid := sess.Get("user_id")
	if uid == nil {
		return c.Redirect("/login")
	}
	c.Locals("user_id", uid)
	return c.Next()
}

func (h *Handler) requireAdmin(c *fiber.Ctx) error {
	uid, _ := c.Locals("user_id").(int64)
	var isAdmin bool
	err := h.db.Pool.QueryRow(c.Context(), `SELECT is_admin FROM users WHERE id = $1`, uid).Scan(&isAdmin)
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
	var id int64
	var hash string
	err := h.db.SQLX.QueryRow(`SELECT id, password FROM users WHERE LOWER(email) = LOWER($1)`, email).Scan(&id, &hash)
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
	_ = sess.Save()
	return c.Redirect("/dashboard")
}

func (h *Handler) Logout(c *fiber.Ctx) error {
	sess, _ := h.sessions.Get(c)
	_ = sess.Destroy()
	return c.Redirect("/login")
}

func (h *Handler) Dashboard(c *fiber.Ctx) error {
	uid, _ := c.Locals("user_id").(int64)
	rows, _ := h.db.Pool.Query(c.Context(), `
		SELECT id, domain_slug, verification_status FROM domains WHERE user_id = $1 ORDER BY id
	`, uid)
	defer rows.Close()
	var b strings.Builder
	b.WriteString(`<div class="card"><h1>Setup</h1><p>Register domains, verify DNS, and manage API keys.</p><ul class="domain-list">`)
	for rows.Next() {
		var id int64
		var slug, status string
		_ = rows.Scan(&id, &slug, &status)
		badge := "badge-pending"
		if status == "verified" || status == "verified_ghl" {
			badge = "badge-verified"
		}
		b.WriteString(`<li><strong>` + web.Esc(slug) + `</strong> <span class="badge ` + badge + `">` + web.Esc(status) + `</span>
<form method="post" action="/dashboard/domains/` + strconv.FormatInt(id, 10) + `/verify-txt"><button type="submit" class="btn btn-sm btn-secondary">Verify TXT</button></form></li>`)
	}
	b.WriteString(`</ul></div>
<div class="card"><h2>API keys</h2><ul class="domain-list">`)
	tokRows, _ := h.db.Pool.Query(c.Context(), `
		SELECT id, name, created_at::text FROM personal_access_tokens
		WHERE tokenable_type = 'App\Models\User' AND tokenable_id = $1
		ORDER BY id DESC
	`, uid)
	if tokRows != nil {
		for tokRows.Next() {
			var tid int64
			var name, created string
			_ = tokRows.Scan(&tid, &name, &created)
			b.WriteString(`<li><strong>` + web.Esc(name) + `</strong> <span class="badge badge-pending">` + web.Esc(created) + `</span>
<form method="post" action="/dashboard/api-tokens/` + strconv.FormatInt(tid, 10) + `/revoke" onsubmit="return confirm('Revoke this token?');">
<button type="submit" class="btn btn-sm btn-secondary">Revoke</button></form></li>`)
		}
		tokRows.Close()
	}
	b.WriteString(`</ul>
<form method="post" action="/dashboard/api-tokens/staging" class="inline-form" style="margin-top:0.75rem">
<button type="submit" class="btn btn-secondary">Create Staging token</button>
</form>
<form method="post" action="/dashboard/api-tokens" class="inline-form">
<label>Token name <input name="name" type="text" placeholder="Production" required></label>
<button type="submit" class="btn btn-primary">Create token</button>
</form></div>
<div class="card"><h2>Add domain</h2>
<form method="post" action="/dashboard/domains" class="inline-form">
<label>Hostname <input name="domain_slug" type="text" placeholder="www.example.com" required></label>
<label>MLS dataset <input name="mls_dataset" type="text" value="stellar"></label>
<button type="submit" class="btn btn-primary">Add domain</button>
</form></div>`)
	var isAdmin bool
	_ = h.db.Pool.QueryRow(c.Context(), `SELECT is_admin FROM users WHERE id = $1`, uid).Scan(&isAdmin)
	if isAdmin {
		b.WriteString(`<div class="card"><h2>Invite user</h2>
<form method="post" action="/dashboard/invitations" class="inline-form">
<label>Email <input name="email" type="email" required></label>
<button type="submit" class="btn btn-primary">Send invitation</button>
</form></div>`)
	}
	return c.Type("html").SendString(web.Page("Dashboard", b.String()))
}

func (h *Handler) StoreDomain(c *fiber.Ctx) error {
	uid, _ := c.Locals("user_id").(int64)
	slug := strings.ToLower(strings.TrimSpace(c.FormValue("domain_slug")))
	mls := c.FormValue("mls_dataset")
	val := randomTXT()
	_, err := h.db.Pool.Exec(c.Context(), `
		INSERT INTO domains (user_id, domain_slug, mls_dataset, allowed_mls_datasets, verification_status,
			txt_verification_name, txt_verification_value, created_at, updated_at)
		VALUES ($1, $2, $3, $4::jsonb, 'pending', $5, $6, NOW(), NOW())
	`, uid, slug, mls, `["`+mls+`"]`, "_quantyra-verify."+slug, val)
	if err != nil {
		return c.Status(400).SendString(err.Error())
	}
	return c.Redirect("/dashboard")
}

func (h *Handler) VerifyTXT(c *fiber.Ctx) error {
	uid, _ := c.Locals("user_id").(int64)
	id := c.Params("id")
	var txtHost, txtVal string
	err := h.db.Pool.QueryRow(c.Context(), `
		SELECT txt_verification_name, txt_verification_value FROM domains WHERE id = $1 AND user_id = $2
	`, id, uid).Scan(&txtHost, &txtVal)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "domain not found")
	}
	ok, err := dns.VerifyTXT(c.Context(), txtHost, txtVal)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "DNS lookup failed")
	}
	if !ok {
		_, _ = h.db.Pool.Exec(c.Context(), `
			UPDATE domains SET verification_checked_at = NOW(), updated_at = NOW() WHERE id = $1
		`, id)
		return c.Status(422).SendString("TXT record not found. Publish the verification record at your DNS host, then try again.")
	}
	_, err = h.db.Pool.Exec(c.Context(), `
		UPDATE domains SET verification_status = 'verified', txt_verified_at = NOW(),
			verification_checked_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, id, uid)
	if err != nil {
		return err
	}
	plain, _ := h.tokens.Create(c.Context(), uid, "Production", []string{"idx:full"})
	body := `<div class="card"><h1>Domain verified</h1><p>Save this production token now — it will not be shown again.</p><div class="token-box" id="token">` + web.Esc(plain) + `</div><p><a class="btn btn-primary" href="/dashboard">Back to dashboard</a></p></div>`
	return c.Type("html").SendString(web.Page("Verified", body))
}

func (h *Handler) CreateStagingToken(c *fiber.Ctx) error {
	uid, _ := c.Locals("user_id").(int64)
	var exists int
	_ = h.db.Pool.QueryRow(c.Context(), `
		SELECT COUNT(*) FROM personal_access_tokens
		WHERE tokenable_type = 'App\Models\User' AND tokenable_id = $1 AND name = 'Staging'
	`, uid).Scan(&exists)
	if exists > 0 {
		return c.Status(409).SendString("Staging token already exists")
	}
	plain, err := h.tokens.Create(c.Context(), uid, "Staging", []string{"idx:full"})
	if err != nil {
		return err
	}
	return c.SendString("Staging token: " + plain)
}

func (h *Handler) CreateToken(c *fiber.Ctx) error {
	uid, _ := c.Locals("user_id").(int64)
	name := c.FormValue("name")
	plain, err := h.tokens.Create(c.Context(), uid, name, []string{"idx:full"})
	if err != nil {
		return err
	}
	return c.SendString("Token: " + plain)
}

func (h *Handler) RevokeToken(c *fiber.Ctx) error {
	uid, _ := c.Locals("user_id").(int64)
	_ = h.tokens.Revoke(c.Context(), uid, parseInt64(c.Params("id")))
	return c.Redirect("/dashboard")
}

func (h *Handler) CreateInvitation(c *fiber.Ctx) error {
	uid, _ := c.Locals("user_id").(int64)
	plain, err := h.invitations.Create(c.Context(), uid, c.FormValue("email"))
	if err != nil {
		return c.Status(400).SendString(err.Error())
	}
	link := "/invite/" + plain
	body := `<div class="card"><h1>Invitation created</h1><p>Share this link (shown once):</p><div class="token-box">` + web.Esc(link) + `</div><p><a class="btn btn-primary" href="/dashboard">Back</a></p></div>`
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

func randomTXT() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func parseInt64(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}
