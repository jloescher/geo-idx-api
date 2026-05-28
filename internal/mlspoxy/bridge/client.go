package bridge

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
	dom "github.com/quantyralabs/idx-api/internal/domain"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/querymerge"
)

// Client proxies Bridge Data Output web + RESO APIs.
type Client struct {
	cfg    config.BridgeConfig
	http   *http.Client
	apiKey string
}

func NewClient(cfg config.Config, feed dom.FeedDefinition) *Client {
	key := cfg.Bridge.APIKey
	return &Client{
		cfg:    cfg.Bridge,
		apiKey: key,
		http: &http.Client{
			Timeout: cfg.Bridge.Timeout,
		},
	}
}

func (c *Client) WebURL(path string, dataset string) string {
	base := strings.TrimRight(c.cfg.Host, "/")
	prefix := strings.Trim(c.cfg.PathPrefix, "/")
	if prefix != "" {
		base += "/" + prefix
	}
	return fmt.Sprintf("%s/%s/%s", base, dataset, strings.TrimLeft(path, "/"))
}

func (c *Client) ResoURL(path string, dataset string) string {
	base := strings.TrimRight(c.cfg.Host, "/")
	parts := []string{base}
	if p := strings.Trim(c.cfg.PathPrefix, "/"); p != "" {
		parts = append(parts, p)
	}
	if r := strings.Trim(c.cfg.ResoRoot, "/"); r != "" {
		parts = append(parts, r)
	}
	parts = append(parts, dataset)
	if path != "" {
		parts = append(parts, strings.TrimLeft(path, "/"))
	}
	return strings.Join(parts, "/")
}

// Proxy forwards the incoming Fiber request to upstream and returns status + body.
func (c *Client) Proxy(fc *fiber.Ctx, upstream string) (int, []byte, http.Header, error) {
	return c.proxy(fc, upstream, true)
}

// ProxyUpstream forwards without merging client query params (search OData URLs).
func (c *Client) ProxyUpstream(fc *fiber.Ctx, upstream string) (int, []byte, http.Header, error) {
	return c.proxy(fc, upstream, false)
}

func (c *Client) proxy(fc *fiber.Ctx, upstream string, mergeQuery bool) (int, []byte, http.Header, error) {
	u, err := url.Parse(upstream)
	if err != nil {
		return 0, nil, nil, err
	}
	if mergeQuery {
		q := u.Query()
		querymerge.IntoUpstream(q, fc.Queries())
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(fc.Context(), fc.Method(), u.String(), requestBody(fc))
	if err != nil {
		return 0, nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	if accept := fc.Get("Accept"); accept != "" {
		req.Header.Set("Accept", accept)
	}
	if ct := fc.Get("Content-Type"); ct != "" {
		req.Header.Set("Content-Type", ct)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return resp.StatusCode, body, resp.Header, err
}

func requestBody(c *fiber.Ctx) io.Reader {
	if c.Method() == fiber.MethodGet || c.Method() == fiber.MethodHead {
		return nil
	}
	b := c.Body()
	if len(b) == 0 {
		return nil
	}
	return bytes.NewReader(b)
}

func DatasetFromFeed(feed dom.FeedDefinition, fallback string) string {
	return dom.DatasetSlug(feed, fallback)
}
