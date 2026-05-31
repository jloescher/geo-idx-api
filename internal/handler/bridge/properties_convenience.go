package bridge

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/mlspoxy"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/upstream"
	"github.com/quantyralabs/idx-api/internal/service/cache"
)

type propertiesConvenienceBody struct {
	City   *string `json:"city"`
	Limit  *int    `json:"limit"`
	Cursor *string `json:"cursor"`
}

type propertiesCursor struct {
	Top  int `json:"top"`
	Skip int `json:"skip"`
}

// PropertiesPost accepts JSON convenience fields and proxies as a RESO Property GET.
func (h *Handler) PropertiesPost(c *fiber.Ctx) error {
	if err := applyPropertiesConvenienceBody(c); err != nil {
		return err
	}
	return h.proxyPropertiesCollection(c)
}

func (h *Handler) proxyPropertiesCollection(c *fiber.Ctx) error {
	applyPropertiesConvenienceQuery(c)
	feed := mlspoxy.Feed(c)
	cli := h.factory.ForRequest(c)
	candidates := upstream.BuildResoCandidates(h.cfg, feed, "Property")
	partition := cache.ResoPartition(h.domainSlug(c), h.feedCode(c), "Property")
	return h.finishProxyMethod(c, "properties.collection", cli, candidates, "", partition, http.MethodGet)
}

func applyPropertiesConvenienceBody(c *fiber.Ctx) error {
	if len(c.Body()) == 0 {
		return nil
	}
	var body propertiesConvenienceBody
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON body")
	}
	mergePropertiesConvenience(c, body)
	return nil
}

func applyPropertiesConvenienceQuery(c *fiber.Ctx) {
	body := propertiesConvenienceBody{}
	if city := strings.TrimSpace(c.Query("city")); city != "" {
		body.City = &city
	}
	if limitStr := strings.TrimSpace(c.Query("limit")); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			body.Limit = &limit
		}
	}
	if cursor := strings.TrimSpace(c.Query("cursor")); cursor != "" {
		body.Cursor = &cursor
	}
	if body.City == nil && body.Limit == nil && body.Cursor == nil {
		return
	}
	mergePropertiesConvenience(c, body)
}

func mergePropertiesConvenience(c *fiber.Ctx, body propertiesConvenienceBody) {
	args := c.Request().URI().QueryArgs()

	if body.City != nil && strings.TrimSpace(*body.City) != "" {
		cityFilter := fmt.Sprintf("City eq '%s'", odataEscape(*body.City))
		if existing := string(args.Peek("$filter")); existing != "" {
			cityFilter = "(" + existing + ") and (" + cityFilter + ")"
		}
		args.Set("$filter", cityFilter)
	}

	if body.Limit != nil && *body.Limit > 0 {
		args.Set("$top", strconv.Itoa(*body.Limit))
	}

	if body.Cursor != nil && strings.TrimSpace(*body.Cursor) != "" {
		if top, skip, ok := decodePropertiesCursor(*body.Cursor); ok {
			if top > 0 {
				args.Set("$top", strconv.Itoa(top))
			}
			if skip > 0 {
				args.Set("$skip", strconv.Itoa(skip))
			}
		}
	}
}

func decodePropertiesCursor(raw string) (top, skip int, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, 0, false
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		b, err = base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return 0, 0, false
		}
	}
	var tok propertiesCursor
	if err := json.Unmarshal(b, &tok); err != nil {
		return 0, 0, false
	}
	if tok.Top <= 0 && tok.Skip <= 0 {
		return 0, 0, false
	}
	return tok.Top, tok.Skip, true
}

func odataEscape(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "'", "''")
}
