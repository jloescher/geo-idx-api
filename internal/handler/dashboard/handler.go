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
	"github.com/quantyralabs/idx-api/internal/web"
)

// Handler provides invite-only dashboard (domains, API keys).
type Handler struct {
	cfg      config.Config
	db       *repository.DB
	tokens   *repository.TokenRepo
	domains  *repository.DomainRepo
	sessions *session.Store
	logger   *slog.Logger
}

func NewHandler(cfg config.Config, db *repository.DB, logger *slog.Logger) *Handler {
	store := session.New(session.Config{
		Expiration: cfg.Auth.SessionLifetime,
	})
	return &Handler{
		cfg:      cfg,
		db:       db,
		tokens:   repository.NewTokenRepo(db),
		domains:  repository.NewDomainRepo(db),
		sessions: store,
		logger:   logger,
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
	app.Delete("/dashboard/api-tokens/:id", h.requireAuth, h.RevokeToken)
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
<div class="card"><h2>Add domain</h2>
<form method="post" action="/dashboard/domains" class="inline-form">
<label>Hostname <input name="domain_slug" type="text" placeholder="www.example.com" required></label>
<label>MLS dataset <input name="mls_dataset" type="text" value="stellar"></label>
<button type="submit" class="btn btn-primary">Add domain</button>
</form></div>
<div class="card"><h2>API keys</h2>
<form method="post" action="/dashboard/api-tokens" class="inline-form">
<label>Token name <input name="name" type="text" placeholder="Production" required></label>
<button type="submit" class="btn btn-primary">Create token</button>
</form></div>`)
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
	_, err := h.db.Pool.Exec(c.Context(), `
		UPDATE domains SET verification_status = 'verified', txt_verified_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, id, uid)
	if err != nil {
		return err
	}
	plain, _ := h.tokens.Create(c.Context(), uid, "Production", []string{"idx:full"})
	body := `<div class="card"><h1>Domain verified</h1><p>Save this production token now — it will not be shown again.</p><div class="token-box" id="token">` + web.Esc(plain) + `</div><p><a class="btn btn-primary" href="/dashboard">Back to dashboard</a></p></div>`
	return c.Type("html").SendString(web.Page("Verified", body))
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

func randomTXT() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func parseInt64(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}
