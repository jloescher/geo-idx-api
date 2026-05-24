package dashboard

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

func bindSessionUser(c *fiber.Ctx, store *session.Store) (int64, *session.Session, bool) {
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
