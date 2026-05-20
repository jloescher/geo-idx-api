package spark

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

// Client proxies Spark Platform live RESO API.
type Client struct {
	cfg   config.SparkConfig
	token string
	http  *http.Client
}

func NewClient(cfg config.Config, _ mls.FeedDefinition) *Client {
	return &Client{
		cfg:   cfg.Spark,
		token: cfg.Spark.AccessToken,
		http:  &http.Client{Timeout: cfg.Spark.Timeout},
	}
}

func (c *Client) LiveResoURL(path, _ string) string {
	base := strings.TrimRight(c.cfg.APIHost, "/")
	ver := strings.Trim(c.cfg.APIVersion, "/")
	root := strings.Trim(c.cfg.LiveResoRoot, "/")
	return fmt.Sprintf("%s/%s/%s/%s", base, ver, root, strings.TrimLeft(path, "/"))
}

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
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return resp.StatusCode, body, resp.Header, err
}
