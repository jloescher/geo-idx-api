package mlspoxy

import (
	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/api/ctxkeys"
	"github.com/quantyralabs/idx-api/internal/config"
	dom "github.com/quantyralabs/idx-api/internal/domain"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/bridge"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/spark"
)

// ProxyClient can forward HTTP to an MLS upstream.
type ProxyClient interface {
	Proxy(c *fiber.Ctx, url string) (int, []byte, map[string][]string, error)
	ProxyMethod(c *fiber.Ctx, url, method string) (int, []byte, map[string][]string, error)
	ProxyUpstream(c *fiber.Ctx, url string) (int, []byte, map[string][]string, error)
}

// Factory selects upstream MLS client by feed provider.
type Factory struct {
	cfg          config.Config
	sparkLimiter spark.RateLimiter
}

func NewFactory(cfg config.Config, sparkLimiter spark.RateLimiter) *Factory {
	return &Factory{cfg: cfg, sparkLimiter: sparkLimiter}
}

func (f *Factory) ForRequest(c *fiber.Ctx) ProxyClient {
	def, _ := c.Locals(ctxkeys.MLSFeedDef).(dom.FeedDefinition)
	if def.Provider == "spark" {
		return &sparkWrapper{spark.NewClient(f.cfg, f.sparkLimiter)}
	}
	return &bridgeWrapper{bridge.NewClient(f.cfg, def)}
}

func Feed(c *fiber.Ctx) dom.FeedDefinition {
	def, _ := c.Locals(ctxkeys.MLSFeedDef).(dom.FeedDefinition)
	return def
}

func FeedCode(c *fiber.Ctx) string {
	code, _ := c.Locals(ctxkeys.MLSFeedCode).(string)
	return code
}

type bridgeWrapper struct{ *bridge.Client }

func (w *bridgeWrapper) Proxy(c *fiber.Ctx, url string) (int, []byte, map[string][]string, error) {
	st, body, hdr, err := w.Client.Proxy(c, url)
	return st, body, hdr, err
}

func (w *bridgeWrapper) ProxyMethod(c *fiber.Ctx, url, method string) (int, []byte, map[string][]string, error) {
	st, body, hdr, err := w.Client.ProxyMethod(c, url, method)
	return st, body, hdr, err
}

func (w *bridgeWrapper) ProxyUpstream(c *fiber.Ctx, url string) (int, []byte, map[string][]string, error) {
	st, body, hdr, err := w.Client.ProxyUpstream(c, url)
	return st, body, hdr, err
}

type sparkWrapper struct{ *spark.Client }

func (w *sparkWrapper) Proxy(c *fiber.Ctx, url string) (int, []byte, map[string][]string, error) {
	st, body, hdr, err := w.Client.Proxy(c, url)
	return st, body, hdr, err
}

func (w *sparkWrapper) ProxyMethod(c *fiber.Ctx, url, method string) (int, []byte, map[string][]string, error) {
	st, body, hdr, err := w.Client.ProxyMethod(c, url, method)
	return st, body, hdr, err
}

func (w *sparkWrapper) ProxyUpstream(c *fiber.Ctx, url string) (int, []byte, map[string][]string, error) {
	st, body, hdr, err := w.Client.ProxyUpstream(c, url)
	return st, body, hdr, err
}
