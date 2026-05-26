package mlspoxy

import (
	"fmt"
	"strings"

	"github.com/quantyralabs/idx-api/internal/config"
	dom "github.com/quantyralabs/idx-api/internal/domain"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/bridge"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/spark"
)

// LiveResoEndpoint is upstream Property collection URL + bearer for live (non-mirror) RESO calls.
type LiveResoEndpoint struct {
	PropertyURL string
	Bearer      string
}

// LiveResoPropertyEndpoint resolves live RESO Property URL and credentials for any catalog feed.
func LiveResoPropertyEndpoint(cfg config.Config, feed dom.FeedDefinition) (LiveResoEndpoint, error) {
	switch feed.Provider {
	case "spark":
		if strings.TrimSpace(cfg.Spark.AccessToken) == "" {
			return LiveResoEndpoint{}, fmt.Errorf("live MLS credentials not configured for feed %q", feed.Code)
		}
		sc := spark.NewClient(cfg, nil)
		return LiveResoEndpoint{
			PropertyURL: sc.LiveResoURL("Property", feed.Dataset),
			Bearer:      cfg.Spark.AccessToken,
		}, nil
	default:
		if strings.TrimSpace(cfg.Bridge.APIKey) == "" {
			return LiveResoEndpoint{}, fmt.Errorf("live MLS credentials not configured for feed %q", feed.Code)
		}
		bc := bridge.NewClient(cfg, feed)
		ds := dom.DatasetSlug(feed, cfg.Bridge.Dataset)
		return LiveResoEndpoint{
			PropertyURL: bc.ResoURL("Property", ds),
			Bearer:      cfg.Bridge.APIKey,
		}, nil
	}
}
