//go:build integration

package search

import (
	"context"
	"testing"
	"time"

	"github.com/xotec-solutions/xotec-datalayer/src/internal/db"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/testutil"
)

func TestSearchCursorPagingIntegration(t *testing.T) {
	pool := testutil.NewPool(t)
	if pool == nil {
		return
	}
	registry := db.NewRegistry()
	ctx := context.Background()

	var hasPostGIS bool
	if err := pool.QueryRow(ctx, registry.SQL(db.QueryName("TestSearchHasPostGIS"))).Scan(&hasPostGIS); err != nil || !hasPostGIS {
		t.Skip("postgis not available")
	}

	listingIDs := []string{"TESTSEARCH-1", "TESTSEARCH-2", "TESTSEARCH-3"}
	for i, listingID := range listingIDs {
		price := float64(300000 + i*50000)
		bedrooms := 2 + i
		bathrooms := 2
		livingArea := float64(1200 + i*200)
		lotSize := float64(0.15 + float64(i)*0.05)
		yearBuilt := 2010 + i
		dom := 5 + i
		status := "Active"
		isActive := true
		mlgCanView := true
		onMarket := time.Now().Add(time.Duration(-i) * 24 * time.Hour)
		city := "Tampa"
		state := "FL"
		postal := "33602"
		streetNum := "100"
		streetName := "Main"
		lat := 27.95 + float64(i)*0.01
		lng := -82.45 - float64(i)*0.01

		var id int64
		err := pool.QueryRow(ctx, registry.SQL(db.QueryName("TestSearchInsertPropertyCore")),
			listingID,
			price,
			bedrooms,
			bathrooms,
			livingArea,
			lotSize,
			yearBuilt,
			dom,
			status,
			isActive,
			mlgCanView,
			onMarket,
			city,
			state,
			postal,
			streetNum,
			streetName,
			lat,
			lng,
		).Scan(&id)
		if err != nil {
			t.Fatalf("insert listing %s: %v", listingID, err)
		}
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, registry.SQL(db.QueryName("TestSearchDeleteListing")), listingIDs)
	})

	store := NewStore(pool, registry, "media.lzrcdn.com")
	req := SearchRequest{
		Params: SearchParams{Sort: "list_price", SortDir: "desc"},
		Page:   PageParams{Limit: IntParam{Value: intValuePtr(2)}},
	}

	res1, err := store.Search(ctx, req, EffectiveLimit(req.Page))
	if err != nil {
		t.Fatalf("search page 1: %v", err)
	}
	if !res1.HasMore || res1.NextCursor == nil {
		t.Fatalf("expected next cursor")
	}
	if len(res1.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(res1.Results))
	}

	req.Page.Cursor = *res1.NextCursor
	res2, err := store.Search(ctx, req, EffectiveLimit(req.Page))
	if err != nil {
		t.Fatalf("search page 2: %v", err)
	}
	if res2.HasMore {
		t.Fatalf("expected no more results")
	}
	if len(res2.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res2.Results))
	}

	seen := map[string]bool{}
	for _, item := range res1.Results {
		seen[item.ListingID] = true
	}
	for _, item := range res2.Results {
		if seen[item.ListingID] {
			t.Fatalf("duplicate listing_id %s", item.ListingID)
		}
	}
}
