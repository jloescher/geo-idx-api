package search

import (
	"context"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/mlspoxy"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/bridge"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// BridgeLiveSearch queries upstream RESO for statuses not in mirror.
type BridgeLiveSearch struct {
	cfg     config.Config
	factory *mlspoxy.Factory
}

func NewBridgeLiveSearch(cfg config.Config, db *repository.DB) *BridgeLiveSearch {
	_ = db
	return &BridgeLiveSearch{cfg: cfg, factory: mlspoxy.NewFactory(cfg)}
}

func (b *BridgeLiveSearch) Search(ctx context.Context, c *fiber.Ctx, feedCode string, req SearchRequest) (SearchResult, error) {
	feed := mlspoxy.Feed(c)
	ds := bridge.DatasetFromFeed(feed, b.cfg.Bridge.Dataset)
	bc := bridge.NewClient(b.cfg, feed)
	upstream := bc.ResoURL("Property", ds)
	status, body, _, err := b.factory.ForRequest(c).Proxy(c, upstream)
	if err != nil || status >= 400 {
		return SearchResult{}, err
	}
	var envelope struct {
		Value []json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return SearchResult{Results: []json.RawMessage{body}}, nil
	}
	return SearchResult{Results: envelope.Value, HasMore: false}, nil
}
