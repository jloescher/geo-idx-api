package bridge

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/service/mls"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client proxies Bridge Data Output web + RESO APIs.
type Client struct {
	cfg    config.BridgeConfig
	http   *http.Client
	apiKey string
}

func NewClient(cfg config.Config, feed mls.FeedDefinition) *Client {
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
	u, err := url.Parse(upstream)
	if err != nil {
		return 0, nil, nil, err
	}
	q := u.Query()
	for k, v := range fc.Queries() {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(fc.Context(), fc.Method(), u.String(), nil)
	if err != nil {
		return 0, nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	if accept := fc.Get("Accept"); accept != "" {
		req.Header.Set("Accept", accept)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return resp.StatusCode, body, resp.Header, err
}

func DatasetFromFeed(feed mls.FeedDefinition, fallback string) string {
	if feed.Dataset != "" {
		return feed.Dataset
	}
	return fallback
}
