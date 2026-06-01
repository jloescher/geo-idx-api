package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Config holds idx-api-web delegation settings.
type Config struct {
	BaseURL     string
	DomainSlug  string
	ServiceToken string
	Timeout     time.Duration
}

// LoadConfigFromEnv reads MCP_API_* environment variables.
func LoadConfigFromEnv() Config {
	timeout := 30 * time.Second
	return Config{
		BaseURL:      strings.TrimRight(os.Getenv("MCP_API_INTERNAL_URL"), "/"),
		DomainSlug:   strings.TrimSpace(os.Getenv("MCP_API_DOMAIN_SLUG")),
		ServiceToken: strings.TrimSpace(os.Getenv("MCP_API_SERVICE_TOKEN")),
		Timeout:      timeout,
	}
}

func (c Config) Enabled() bool {
	return c.BaseURL != "" && c.ServiceToken != ""
}

// Client calls idx-api-web /api/v1 routes with service PAT auth.
type Client struct {
	cfg    Config
	client *http.Client
}

func New(cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &Client{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *Client) Get(ctx context.Context, path string, query url.Values) (json.RawMessage, int, error) {
	if !c.cfg.Enabled() {
		return nil, 0, fmt.Errorf("MCP API client not configured (set MCP_API_INTERNAL_URL and MCP_API_SERVICE_TOKEN)")
	}
	u := c.cfg.BaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, err
	}
	c.applyHeaders(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		return body, resp.StatusCode, fmt.Errorf("upstream HTTP %d: %s", resp.StatusCode, truncate(string(body), 512))
	}
	return body, resp.StatusCode, nil
}

func (c *Client) Post(ctx context.Context, path string, query url.Values, payload any) (json.RawMessage, int, error) {
	if !c.cfg.Enabled() {
		return nil, 0, fmt.Errorf("MCP API client not configured (set MCP_API_INTERNAL_URL and MCP_API_SERVICE_TOKEN)")
	}
	u := c.cfg.BaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyHeaders(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		return respBody, resp.StatusCode, fmt.Errorf("upstream HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 512))
	}
	return respBody, resp.StatusCode, nil
}

func (c *Client) applyHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.cfg.ServiceToken)
	if c.cfg.DomainSlug != "" {
		req.Header.Set("X-Domain-Slug", c.cfg.DomainSlug)
	}
}

func (c *Client) Enabled() bool {
	return c.cfg.Enabled()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
