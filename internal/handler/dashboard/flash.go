package dashboard

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	dashsvc "github.com/quantyralabs/idx-api/internal/service/dashboard"
)

const (
	provisionFlashKey  = "provision_flash"
	tokenRevealFlashKey = "token_reveal_flash"
)

// tokenRevealFlash maps domain ID string to one-time plaintext token.
type tokenRevealFlash map[string]string

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

func (h *Handler) setTokenReveal(c *fiber.Ctx, domainID int64, plain string) error {
	sess, err := h.sessions.Get(c)
	if err != nil {
		return err
	}
	var m tokenRevealFlash
	if raw, ok := sess.Get(tokenRevealFlashKey).(string); ok && raw != "" {
		_ = json.Unmarshal([]byte(raw), &m)
	}
	if m == nil {
		m = tokenRevealFlash{}
	}
	m[fmt.Sprintf("%d", domainID)] = plain
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	sess.Set(tokenRevealFlashKey, string(b))
	return sess.Save()
}

func (h *Handler) takeTokenReveals(c *fiber.Ctx) map[int64]string {
	sess, err := h.sessions.Get(c)
	if err != nil {
		return nil
	}
	raw, ok := sess.Get(tokenRevealFlashKey).(string)
	if !ok || raw == "" {
		return nil
	}
	sess.Delete(tokenRevealFlashKey)
	_ = sess.Save()
	var m tokenRevealFlash
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil
	}
	out := make(map[int64]string, len(m))
	for k, v := range m {
		id, err := strconv.ParseInt(k, 10, 64)
		if err == nil && v != "" {
			out[id] = v
		}
	}
	return out
}
