package mlspoxy

import (
	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/api/ctxkeys"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/bridge"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/spark"
	"github.com/quantyralabs/idx-api/internal/service/mls"
)

// ProxyClient can forward HTTP to an MLS upstream.
type ProxyClient interface {
	Proxy(c *fiber.Ctx, url string) (int, []byte, map[string][]string, error)
}

// Factory selects Bridge vs Spark client by feed.
type Factory struct {
	cfg config.Config
}

func NewFactory(cfg config.Config) *Factory {
	return &Factory{cfg: cfg}
}

func (f *Factory) ForRequest(c *fiber.Ctx) ProxyClient {
	def, _ := c.Locals(ctxkeys.MLSFeedDef).(mls.FeedDefinition)
	if def.Provider == "spark" {
		return &sparkWrapper{spark.NewClient(f.cfg, def)}
	}
	return &bridgeWrapper{bridge.NewClient(f.cfg, def)}
}

func Feed(c *fiber.Ctx) mls.FeedDefinition {
	def, _ := c.Locals(ctxkeys.MLSFeedDef).(mls.FeedDefinition)
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

type sparkWrapper struct{ *spark.Client }

func (w *sparkWrapper) Proxy(c *fiber.Ctx, url string) (int, []byte, map[string][]string, error) {
	st, body, hdr, err := w.Client.Proxy(c, url)
	return st, body, hdr, err
}
