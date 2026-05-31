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

// PubURL builds Bridge public-records paths.
// Pub endpoints live at the host root (/pub/…) without any version prefix —
// BRIDGE_PATH_PREFIX (e.g. "api/v2") applies only to the Web and RESO APIs.
func (c *Client) PubURL(path string) string {
	base := strings.TrimRight(c.cfg.Host, "/")
	return base + "/" + strings.TrimLeft(path, "/")
}

// ResoBase returns the OData collection base for a dataset (matches replication sync URLs).
func (c *Client) ResoBase(dataset string) string {
	host := strings.TrimRight(c.cfg.Host, "/")
	prefix := strings.Trim(c.cfg.PathPrefix, "/")
	resoRoot := strings.Trim(c.cfg.ResoRoot, "/")

	var basePath string
	switch {
	case prefix != "":
		basePath = prefix + "/OData/" + dataset
	case resoRoot != "":
		basePath = resoRoot + "/OData/" + dataset
	default:
		basePath = "OData/" + dataset
	}
	return host + "/" + basePath
}

// ResoURL builds a RESO OData entity URL under ResoBase.
func (c *Client) ResoURL(entity, dataset string) string {
	return c.ResoBase(dataset) + "/" + strings.TrimLeft(entity, "/")
}

// LegacyResoURL builds the pre-P0 path shape for 404 fallback (prefix/resoRoot/dataset/entity).
func (c *Client) LegacyResoURL(entity, dataset string) string {
	host := strings.TrimRight(c.cfg.Host, "/")
	parts := []string{host}
	if p := strings.Trim(c.cfg.PathPrefix, "/"); p != "" {
		parts = append(parts, p)
	}
	if r := strings.Trim(c.cfg.ResoRoot, "/"); r != "" {
		parts = append(parts, r)
	}
	parts = append(parts, dataset)
	if entity != "" {
		parts = append(parts, strings.TrimLeft(entity, "/"))
	}
	return strings.Join(parts, "/")
}

// BareResoURL builds OData paths without path prefix (no-prefix deployments).
func (c *Client) BareResoURL(entity, dataset string) string {
	host := strings.TrimRight(c.cfg.Host, "/")
	url := host + "/OData/" + dataset
	if entity != "" {
		url += "/" + strings.TrimLeft(entity, "/")
	}
	return url
}

// Proxy forwards the incoming Fiber request to upstream and returns status + body.
func (c *Client) Proxy(fc *fiber.Ctx, upstream string) (int, []byte, http.Header, error) {
	return c.proxyWithMethod(fc, upstream, true, fc.Method())
}

// ProxyMethod forwards using an explicit upstream HTTP method (ignores inbound method/body for GET/HEAD).
func (c *Client) ProxyMethod(fc *fiber.Ctx, upstream, method string) (int, []byte, http.Header, error) {
	if method == "" {
		method = fc.Method()
	}
	return c.proxyWithMethod(fc, upstream, true, method)
}

// ProxyUpstream forwards without merging client query params (search OData URLs).
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

func DatasetFromFeed(feed dom.FeedDefinition, fallback string) string {
	return dom.DatasetSlug(feed, fallback)
}
