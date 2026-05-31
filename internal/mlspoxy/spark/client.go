package spark

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/querymerge"
)

// RateLimiter coordinates outbound Spark HTTP across API and worker processes.
type RateLimiter interface {
	Wait(ctx context.Context) error
}

// Client proxies Spark Platform live RESO API.
type Client struct {
	cfg     config.SparkConfig
	token   string
	http    *http.Client
	limiter RateLimiter
}

func NewClient(cfg config.Config, limiter RateLimiter) *Client {
	return &Client{
		cfg:     cfg.Spark,
		token:   cfg.Spark.AccessToken,
		http:    &http.Client{Timeout: cfg.Spark.Timeout},
		limiter: limiter,
	}
}

func requestBodyForMethod(c *fiber.Ctx, method string) io.Reader {
	if method == http.MethodGet || method == http.MethodHead {
		return nil
	}
	b := c.Body()
	if len(b) == 0 {
		return nil
	}
	return bytes.NewReader(b)
}

func (c *Client) LiveResoURL(path, _ string) string {
	base := strings.TrimRight(c.cfg.APIHost, "/")
	ver := strings.Trim(c.cfg.APIVersion, "/")
	root := strings.Trim(c.cfg.LiveResoRoot, "/")
	return fmt.Sprintf("%s/%s/%s/%s", base, ver, root, strings.TrimLeft(path, "/"))
}

// WebURL builds the Spark native listings web API URL.
func (c *Client) WebURL(path string) string {
	base := strings.TrimRight(c.cfg.APIHost, "/")
	ver := strings.Trim(c.cfg.APIVersion, "/")
	return fmt.Sprintf("%s/%s/%s", base, ver, strings.TrimLeft(path, "/"))
}

// ResoV3URL is the Spark v3 RESO OData fallback when v1 returns 404.
func (c *Client) ResoV3URL(path string) string {
	base := strings.TrimRight(c.cfg.APIHost, "/")
	ver := strings.Trim(c.cfg.APIVersion, "/")
	return fmt.Sprintf("%s/%s/Version/3/Reso/OData/%s", base, ver, strings.TrimLeft(path, "/"))
}

func (c *Client) Proxy(fc *fiber.Ctx, upstream string) (int, []byte, http.Header, error) {
	return c.proxyWithMethod(fc, upstream, true, fc.Method())
}

func (c *Client) ProxyMethod(fc *fiber.Ctx, upstream, method string) (int, []byte, http.Header, error) {
	if method == "" {
		method = fc.Method()
	}
	return c.proxyWithMethod(fc, upstream, true, method)
}

func (c *Client) ProxyUpstream(fc *fiber.Ctx, upstream string) (int, []byte, http.Header, error) {
	return c.proxyWithMethod(fc, upstream, false, fc.Method())
}

func (c *Client) proxyWithMethod(fc *fiber.Ctx, upstream string, mergeQuery bool, method string) (int, []byte, http.Header, error) {
	u, err := url.Parse(upstream)
	if err != nil {
		return 0, nil, nil, err
	}
	if mergeQuery {
		q := u.Query()
		querymerge.IntoUpstream(q, fc.Queries())
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(fc.Context(), method, u.String(), requestBodyForMethod(fc, method))
	if err != nil {
		return 0, nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if ct := fc.Get("Content-Type"); ct != "" {
		req.Header.Set("Content-Type", ct)
	}

	if c.limiter != nil {
		if err := c.limiter.Wait(fc.Context()); err != nil {
			return 0, nil, nil, err
		}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return resp.StatusCode, body, resp.Header, err
}
