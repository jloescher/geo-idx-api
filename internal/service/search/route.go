package search

import "encoding/json"

// HybridSearchRouteMode determines data source for search.
type HybridSearchRouteMode int

const (
	RoutePostgresOnly HybridSearchRouteMode = iota
	RouteUpstreamOnly
	RouteSplit
)

// DecideRoute implements HybridReplicaSearchDecision parity.
func DecideRoute(req SearchRequest) HybridSearchRouteMode {
	if req.PriceReducedWithinDays != nil && *req.PriceReducedWithinDays > 0 {
		return RouteUpstreamOnly
	}
	hasAP, hasClosed := false, false
	for _, st := range req.Statuses {
		switch st {
		case "Active", "Pending":
			hasAP = true
		case "Closed":
			hasClosed = true
		}
	}
	if len(req.Statuses) == 0 {
		if req.ActiveOnly != nil && !*req.ActiveOnly {
			return RouteUpstreamOnly
		}
		return RoutePostgresOnly
	}
	if hasAP && hasClosed {
		return RouteSplit
	}
	if hasClosed && !hasAP {
		return RouteUpstreamOnly
	}
	if hasAP && !hasClosed {
		return RoutePostgresOnly
	}
	return RouteUpstreamOnly
}

func MergeResults(a, b SearchResult) SearchResult {
	seen := map[string]struct{}{}
	var out []json.RawMessage
	for _, r := range a.Results {
		out = append(out, r)
		seen[string(r)] = struct{}{}
	}
	for _, r := range b.Results {
		if _, ok := seen[string(r)]; !ok {
			out = append(out, r)
		}
	}
	return SearchResult{Results: out, HasMore: a.HasMore || b.HasMore, NextSkip: a.NextSkip}
}
