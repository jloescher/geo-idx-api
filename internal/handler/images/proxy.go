package images

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	db      *repository.DB
	factory *mlspoxy.Factory
}

func NewHandler(cfg config.Config, db *repository.DB, _ *slog.Logger) *Handler {
	return &Handler{cfg: cfg, db: db, factory: mlspoxy.NewFactory(cfg)}
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
	cli := h.factory.ForRequest(c)
	upstream := ""
	if feed.Provider == "spark" {
		upstream = h.sparkMediaURL(c.Context(), ds, listingKey, photoID)
	}
	if upstream == "" {
		upstream = strings.ReplaceAll(strings.ReplaceAll(h.cfg.Bridge.ListingPhotoPath, "{dataset}", ds), "{listingKey}", listingKey)
		upstream = strings.ReplaceAll(upstream, "{photoId}", photoID)
		if !strings.HasPrefix(upstream, "http") {
			upstream = strings.TrimRight(h.cfg.Bridge.Host, "/") + upstream
		}
	}
	status, body, hdr, err := cli.Proxy(c, upstream)
	if err != nil || status >= 400 {
		return fiber.NewError(fiber.StatusBadGateway, "image upstream error")
	}
	_ = os.MkdirAll(filepath.Dir(cachePath), 0o755)
	_ = os.WriteFile(cachePath, body, 0o644)
	ct := "image/jpeg"
	if v := hdr["Content-Type"]; len(v) > 0 {
		ct = v[0]
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

func (h *Handler) sparkMediaURL(ctx context.Context, dataset, listingKey, photoID string) string {
	var media []byte
	err := h.db.Pool.QueryRow(ctx, `
		SELECT media FROM listings WHERE dataset_slug = $1 AND listing_key = $2
	`, dataset, listingKey).Scan(&media)
	if err != nil || len(media) == 0 {
		return ""
	}
	var items []map[string]any
	if json.Unmarshal(media, &items) != nil {
		return ""
	}
	for _, item := range items {
		id, _ := item["MediaKey"].(string)
		if id == "" {
			if n, ok := item["MediaKey"].(float64); ok {
				id = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(fmt.Sprintf("%.0f", n), ".0"), ".0"))
			}
		}
		if id != photoID && fmt.Sprint(item["MediaKey"]) != photoID {
			continue
		}
		if u, ok := item["MediaURL"].(string); ok && u != "" {
			return u
		}
	}
	return ""
}

// Ensure ctxkeys used for domain auth on image routes
var _ = ctxkeys.MLSDomainSlug
