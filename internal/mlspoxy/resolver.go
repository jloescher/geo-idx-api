package mlspoxy

import (
	"github.com/quantyralabs/idx-api/internal/config"
	dom "github.com/quantyralabs/idx-api/internal/domain"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/bridge"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/spark"
)

// UpstreamResolver builds per-feed MLS upstream URLs.
type UpstreamResolver struct {
	cfg config.Config
}

func NewUpstreamResolver(cfg config.Config) *UpstreamResolver {
	return &UpstreamResolver{cfg: cfg}
}

func (r *UpstreamResolver) WebURL(feed dom.FeedDefinition, path string) string {
	if feed.Provider == "spark" {
		return spark.NewClient(r.cfg, nil).WebURL(path)
	}
	ds := dom.DatasetSlug(feed, r.cfg.Bridge.Dataset)
	return bridge.NewClient(r.cfg, feed).WebURL(path, ds)
}

func (r *UpstreamResolver) ResoURL(feed dom.FeedDefinition, entity string) string {
	if feed.Provider == "spark" {
		return spark.NewClient(r.cfg, nil).LiveResoURL(entity, feed.Dataset)
	}
	ds := dom.DatasetSlug(feed, r.cfg.Bridge.Dataset)
	return bridge.NewClient(r.cfg, feed).ResoURL(entity, ds)
}

func (r *UpstreamResolver) PubURL(path string) string {
	return bridge.NewClient(r.cfg, dom.FeedDefinition{Provider: "bridge"}).PubURL(path)
}
