package dashboard

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	dashsvc "github.com/quantyralabs/idx-api/internal/service/dashboard"
)

const provisionFlashKey = "provision_flash"

func (h *Handler) setProvisionFlash(c *fiber.Ctx, result *dashsvc.ProvisionResult) error {
	sess, err := h.sessions.Get(c)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	sess.Set(provisionFlashKey, string(raw))
	return sess.Save()
}

func (h *Handler) takeProvisionFlash(c *fiber.Ctx) *dashsvc.ProvisionResult {
	sess, err := h.sessions.Get(c)
	if err != nil {
		return nil
	}
	raw, ok := sess.Get(provisionFlashKey).(string)
	if !ok || raw == "" {
		return nil
	}
	sess.Delete(provisionFlashKey)
	_ = sess.Save()
	var result dashsvc.ProvisionResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil
	}
	return &result
}
