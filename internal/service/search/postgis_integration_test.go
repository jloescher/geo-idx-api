//go:build integration

package search

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/quantyralabs/idx-api/internal/repository"
)

func TestPostgisSearch_LargoFilters_integration(t *testing.T) {
	dsn := os.Getenv("GOOSE_DBSTRING")
	if dsn == "" {
		t.Skip("GOOSE_DBSTRING not set")
	}
	ctx := context.Background()
	db, err := repository.NewFromDSN(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	minPrice := 250000.0
	minBeds := 2
	city := "Largo"
	lowRisk := true
	maxFees := 500.0
	limit := 24
	req := SearchRequest{
		MinPrice:         &minPrice,
		BedsMin:          &minBeds,
		City:             &city,
		Statuses:         []string{"Active"},
		LowRiskFloodzone: &lowRisk,
		MaxMonthlyFees:   &maxFees,
		Limit:            limit,
	}

	p := NewPostgisSearch(db)
	res, err := p.Search(ctx, "bridge_stellar", req, 12)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	t.Logf("results=%d hasMore=%v nextSkip=%d", len(res.Results), res.HasMore, res.NextSkip)
	for i, r := range res.Results {
		if !json.Valid(r) {
			t.Fatalf("result %d invalid json", i)
		}
	}
}
