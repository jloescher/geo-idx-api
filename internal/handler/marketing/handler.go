package marketing

import (
	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/web"
)

// Handler serves marketing home.
type Handler struct {
	cfg config.Config
}

func NewHandler(cfg config.Config) *Handler {
	return &Handler{cfg: cfg}
}

func (h *Handler) Home(c *fiber.Ctx) error {
	body := `<section class="hero">
<h1>Quantyra IDX</h1>
<p>MLS proxy, image delivery, and developer setup for your IDX sites.</p>
<div class="hero-actions">
<a class="btn btn-primary" href="/dashboard">Open dashboard</a>
<a class="btn btn-secondary" href="/login">Sign in</a>
</div>
</section>`
	return c.Type("html").SendString(web.Page("Home", body))
}
