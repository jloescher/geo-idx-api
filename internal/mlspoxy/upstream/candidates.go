package upstream

import (
	"github.com/quantyralabs/idx-api/internal/config"
	dom "github.com/quantyralabs/idx-api/internal/domain"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/bridge"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/spark"
)

// Candidate is one upstream URL to try in order.
type Candidate struct {
	URL string
	Leg string
}

// BuildResoCandidates returns ordered RESO OData URLs for proxy routes (404 retry only).
func BuildResoCandidates(cfg config.Config, feed dom.FeedDefinition, entity string) []Candidate {
	switch feed.Provider {
	case "spark":
		sc := spark.NewClient(cfg, nil)
		out := []Candidate{{URL: sc.LiveResoURL(entity, feed.Dataset), Leg: "reso"}}
		v3 := sc.ResoV3URL(entity)
		if v3 != out[0].URL {
			out = append(out, Candidate{URL: v3, Leg: "reso-v3"})
		}
		return out
	default:
		bc := bridge.NewClient(cfg, feed)
		ds := dom.DatasetSlug(feed, cfg.Bridge.Dataset)
		primary := bc.ResoURL(entity, ds)
		out := []Candidate{{URL: primary, Leg: "reso"}}
		legacy := bc.LegacyResoURL(entity, ds)
		if legacy != primary {
			out = append(out, Candidate{URL: legacy, Leg: "reso-legacy"})
		}
		bare := bc.BareResoURL(entity, ds)
		if bare != primary && bare != legacy {
			out = append(out, Candidate{URL: bare, Leg: "reso-bare"})
		}
		return out
	}
}

// BuildPropertySearchCandidates returns RESO Property URLs plus Web listings fallback for hybrid search.
func BuildPropertySearchCandidates(cfg config.Config, feed dom.FeedDefinition) []Candidate {
	out := BuildResoCandidates(cfg, feed, "Property")
	switch feed.Provider {
	case "spark":
		sc := spark.NewClient(cfg, nil)
		out = append(out, Candidate{URL: sc.WebURL("listings"), Leg: "web"})
	default:
		bc := bridge.NewClient(cfg, feed)
		ds := dom.DatasetSlug(feed, cfg.Bridge.Dataset)
		out = append(out, Candidate{URL: bc.WebURL("listings", ds), Leg: "web"})
	}
	return out
}

// SingleURL wraps one upstream URL as a single-candidate chain.
func SingleURL(url, leg string) []Candidate {
	return []Candidate{{URL: url, Leg: leg}}
}
