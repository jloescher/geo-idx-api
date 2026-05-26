package dashboard

import (
	"github.com/gofiber/fiber/v2"
)

// SessionAuthMiddleware requires a dashboard session and returns JSON 401 when missing.
func (h *Handler) SessionAuthMiddleware(c *fiber.Ctx) error {
	uid, _, ok := bindSessionUser(c, h.sessions)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	c.Locals("user_id", uid)
	return c.Next()
}

// MonitoringJSON returns the live monitoring snapshot for dashboard JS and admin API.
func (h *Handler) MonitoringJSON(c *fiber.Ctx) error {
	snap, err := h.monitoring.BuildSnapshot(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(snap)
}
