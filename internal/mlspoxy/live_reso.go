package mlspoxy

import (
	"github.com/quantyralabs/idx-api/internal/config"
	dom "github.com/quantyralabs/idx-api/internal/domain"
)

// LiveResoEndpoint is upstream Property collection URL + bearer for live (non-mirror) RESO calls.
type LiveResoEndpoint struct {
	PropertyURL string
	Bearer      string
}

// LiveResoPropertyEndpoint resolves live RESO Property URL and credentials for any catalog feed.
func LiveResoPropertyEndpoint(cfg config.Config, feed dom.FeedDefinition) (LiveResoEndpoint, error) {
	ep, err := ResolveLivePropertyEndpoint(cfg, feed)
	if err != nil {
		return LiveResoEndpoint{}, err
	}
	return LiveResoEndpoint{PropertyURL: ep.ResoURL, Bearer: ep.Bearer}, nil
}
