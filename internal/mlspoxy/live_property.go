package mlspoxy

import (
	"fmt"
	"strings"

	"github.com/quantyralabs/idx-api/internal/config"
	dom "github.com/quantyralabs/idx-api/internal/domain"
)

// LivePropertyEndpoint is upstream Property collection URLs for hybrid search.
type LivePropertyEndpoint struct {
	ResoURL  string
	WebURL   string
	Bearer   string
	Provider string
}

// ResolveLivePropertyEndpoint resolves live Property URLs and credentials for any feed.
func ResolveLivePropertyEndpoint(cfg config.Config, feed dom.FeedDefinition) (LivePropertyEndpoint, error) {
	res := NewUpstreamResolver(cfg)
	switch feed.Provider {
	case "spark":
		if strings.TrimSpace(cfg.Spark.AccessToken) == "" {
			return LivePropertyEndpoint{}, fmt.Errorf("live MLS credentials not configured for feed %q", feed.Code)
		}
		return LivePropertyEndpoint{
			ResoURL:  res.ResoURL(feed, "Property"),
			WebURL:   res.WebURL(feed, "listings"),
			Bearer:   cfg.Spark.AccessToken,
			Provider: "spark",
		}, nil
	default:
		if strings.TrimSpace(cfg.Bridge.APIKey) == "" {
			return LivePropertyEndpoint{}, fmt.Errorf("live MLS credentials not configured for feed %q", feed.Code)
		}
		return LivePropertyEndpoint{
			ResoURL:  res.ResoURL(feed, "Property"),
			WebURL:   res.WebURL(feed, "listings"),
			Bearer:   cfg.Bridge.APIKey,
			Provider: "bridge",
		}, nil
	}
}
