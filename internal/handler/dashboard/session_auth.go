package dashboard

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

// BindSessionUser extracts the user_id from the dashboard session (if present)
// and returns it. It is exported so other packages (e.g. the OAuth handler
// registration) can bind the session for routes that need to know if the user
// is logged in (consent screen, etc.).
func BindSessionUser(c *fiber.Ctx, store *session.Store) (int64, *session.Session, bool) {
	sess, err := store.Get(c)
	if err != nil {
		return 0, nil, false
	}
	uid, ok := parseSessionUserID(sess.Get("user_id"))
	if !ok {
		return 0, sess, false
	}
	_ = sess.Save()
	return uid, sess, true
}

// bindSessionUser is the internal unexported name kept for backward compat
// inside the dashboard package.
func bindSessionUser(c *fiber.Ctx, store *session.Store) (int64, *session.Session, bool) {
	return BindSessionUser(c, store)
}

func parseSessionUserID(raw any) (int64, bool) {
	switch v := raw.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case uint64:
		return int64(v), true
	case uint:
		return int64(v), true
	case float64:
		return int64(v), true
	case float32:
		return int64(v), true
	case string:
		if v == "" {
			return 0, false
		}
		n, err := strconv.ParseInt(v, 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}
