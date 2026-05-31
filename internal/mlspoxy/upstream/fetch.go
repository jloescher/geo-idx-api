package upstream

import (
	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/mlspoxy"
)

const maxCandidates = 3

// FetchResult is the outcome of an upstream fetch with optional fallback.
type FetchResult struct {
	Status int
	Body   []byte
	Header map[string][]string
	Leg    string
	URL    string
}

// FetchWithFallback tries candidates in order; retries only on HTTP 404.
func FetchWithFallback(c *fiber.Ctx, cli mlspoxy.ProxyClient, candidates []Candidate) (FetchResult, error) {
	if len(candidates) == 0 {
		return FetchResult{}, fiber.NewError(fiber.StatusBadGateway, "no upstream candidates")
	}
	if len(candidates) > maxCandidates {
		candidates = candidates[:maxCandidates]
	}

	var last FetchResult
	for i, cand := range candidates {
		status, body, hdr, err := cli.Proxy(c, cand.URL)
		if err != nil {
			return FetchResult{}, err
		}
		last = FetchResult{
			Status: status,
			Body:   body,
			Header: hdr,
			Leg:    cand.Leg,
			URL:    cand.URL,
		}
		if status != fiber.StatusNotFound || i == len(candidates)-1 {
			return last, nil
		}
	}
	return last, nil
}
