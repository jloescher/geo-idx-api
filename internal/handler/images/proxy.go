package images

import (
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/api/ctxkeys"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/mlspoxy"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/bridge"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// Handler streams MLS listing photos with NVMe disk cache.
// Revenue impact: edge-cacheable images improve LCP on listing pages (conversion).
type Handler struct {
	cfg     config.Config
	factory *mlspoxy.Factory
}

func NewHandler(cfg config.Config, _ *repository.DB, _ *slog.Logger) *Handler {
	return &Handler{cfg: cfg, factory: mlspoxy.NewFactory(cfg)}
}

func (h *Handler) Show(c *fiber.Ctx) error {
	listingKey := c.Params("listingKey")
	photoID := c.Params("photoId")
	cachePath := h.cacheFile(listingKey, photoID)
	if data, err := os.ReadFile(cachePath); err == nil {
		c.Set("Cache-Control", "public, max-age=31536000, immutable")
		c.Set("X-IDX-Proxied-Public-Url", h.publicURL(listingKey, photoID))
		return c.Type("image/jpeg").Send(data)
	}
	feed := mlspoxy.Feed(c)
	ds := bridge.DatasetFromFeed(feed, h.cfg.Bridge.Dataset)
	bc := bridge.NewClient(h.cfg, feed)
	upstream := strings.ReplaceAll(strings.ReplaceAll(h.cfg.Bridge.ListingPhotoPath, "{dataset}", ds), "{listingKey}", listingKey)
	upstream = strings.ReplaceAll(upstream, "{photoId}", photoID)
	if !strings.HasPrefix(upstream, "http") {
		upstream = strings.TrimRight(h.cfg.Bridge.Host, "/") + upstream
	}
	status, body, hdr, err := bc.Proxy(c, upstream)
	if err != nil || status >= 400 {
		return fiber.NewError(fiber.StatusBadGateway, "image upstream error")
	}
	_ = os.MkdirAll(filepath.Dir(cachePath), 0o755)
	_ = os.WriteFile(cachePath, body, 0o644)
	ct := hdr.Get("Content-Type")
	if ct == "" {
		ct = "image/jpeg"
	}
	c.Set("Content-Type", ct)
	c.Set("Cache-Control", "public, max-age=31536000, immutable")
	c.Set("X-IDX-Proxied-Public-Url", h.publicURL(listingKey, photoID))
	return c.Send(body)
}

func (h *Handler) cacheFile(listingKey, photoID string) string {
	sum := sha256.Sum256([]byte(listingKey + "/" + photoID))
	name := hex.EncodeToString(sum[:]) + ".bin"
	return filepath.Join(h.cfg.Images.Path, name[:2], name)
}

func (h *Handler) publicURL(listingKey, photoID string) string {
	return strings.TrimRight(h.cfg.Idx.ImagesPublic, "/") + "/images/" + listingKey + "/" + photoID
}

// Ensure ctxkeys used for domain auth on image routes
var _ = ctxkeys.BridgeDomainSlug
