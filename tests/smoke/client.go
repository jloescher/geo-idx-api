//go:build smoke

package smoke

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPResponse holds the result of a smoke HTTP call.
type HTTPResponse struct {
	StatusCode  int
	Body        []byte
	ContentType string
	URL         string
}

// Client performs authenticated smoke requests against the API.
type Client struct {
	cfg   Config
	http  *http.Client
	token string
}

func NewClient(cfg Config) *Client {
	token := cfg.BearerToken
	return &Client{
		cfg:   cfg,
		token: token,
		http: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) Do(method, path string, query map[string]string, headers map[string]string, body []byte, auth bool) (HTTPResponse, error) {
	u, err := url.Parse(c.cfg.BaseURL + path)
	if err != nil {
		return HTTPResponse{}, err
	}
	if len(query) > 0 {
		q := u.Query()
		for k, v := range query {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, u.String(), bodyReader)
	if err != nil {
		return HTTPResponse{}, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if auth && c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("X-Domain-Slug", c.cfg.DomainSlug)
	}
	if c.cfg.SessionCookie != "" {
		req.Header.Set("Cookie", "session_id="+c.cfg.SessionCookie)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return HTTPResponse{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return HTTPResponse{}, err
	}
	return HTTPResponse{
		StatusCode:  resp.StatusCode,
		Body:        respBody,
		ContentType: resp.Header.Get("Content-Type"),
		URL:         u.String(),
	}, nil
}

func (c *Client) Login(email, password string) (string, error) {
	payload := fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)
	resp, err := c.Do(http.MethodPost, "/api/auth/token", nil, map[string]string{
		"Content-Type": "application/json",
	}, []byte(payload), false)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login status %d: %s", resp.StatusCode, truncate(string(resp.Body), 200))
	}
	token, ok := jsonStringAt(resp.Body, "token")
	if !ok || token == "" {
		return "", fmt.Errorf("login response missing token")
	}
	return token, nil
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
