//go:build integration

package search

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/db"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/testutil"
)

// TestSearchIncludesAliasMatchedCities verifies that when a user selects a canonical city
// from autocomplete, the search results include ALL properties normalized to that city,
// including properties that were originally listed with alternate names (aliases).
func TestSearchIncludesAliasMatchedCities(t *testing.T) {
	pool := testutil.NewPool(t)
	if pool == nil {
		t.Skip("test DB not configured")
		return
	}
	registry := db.NewRegistry()
	ctx := context.Background()

	// Setup: Create state
	var stateID int64
	err := pool.QueryRow(ctx, `
		INSERT INTO states (name, abbrev, normalized_name, slug, is_authoritative)
		VALUES ('Test State', 'TS', 'test state', 'test-state', true)
		RETURNING id
	`).Scan(&stateID)
	require.NoError(t, err, "failed to create test state")

	// Setup: Create county
	var countyID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO counties (name, normalized_name, slug, state_id, is_authoritative)
		VALUES ('Test County', 'test county', 'test-county', $1, true)
		RETURNING id
	`, stateID).Scan(&countyID)
	require.NoError(t, err, "failed to create test county")

	// Setup: Create canonical city
	var canonicalCityID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO cities (name, normalized_name, slug, state_id, county_id, is_authoritative)
		VALUES ('San Antonio', 'san antonio', 'san-antonio', $1, $2, true)
		RETURNING id
	`, stateID, countyID).Scan(&canonicalCityID)
	require.NoError(t, err, "failed to create canonical city")

	// Setup: Create matched alias
	var aliasID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO city_aliases (raw_name, normalized_name, state_id, county_id, city_id, status, confidence_score, match_count)
		VALUES ('SA', 'sa', $1, $2, $3, 'matched', 0.95, 100)
		RETURNING id
	`, stateID, countyID, canonicalCityID).Scan(&aliasID)
	require.NoError(t, err, "failed to create city alias")

	// Setup: Seed property matched via canonical name
	onMarket := time.Now().Add(-24 * time.Hour)
	_, err = pool.Exec(ctx, `
		INSERT INTO properties_core (
			listing_id, list_price, bedrooms_total, bathrooms_total, living_area, lot_size,
			year_built, days_on_market, standard_status, is_active, mlg_can_view,
			on_market_date, city, state, postal_code, street_number, street_name,
			latitude, longitude, partition_group,
			city_ref_id, state_ref_id
		) VALUES (
			'PROP-CANONICAL-001', 300000, 3, 2, 1500, 0.25,
			2015, 10, 'Active', true, true,
			$1, 'San Antonio', 'TS', '12345', '100', 'Main St',
			29.4241, -98.4936, 'active',
			$2, $3
		)
	`, onMarket, canonicalCityID, stateID)
	require.NoError(t, err, "failed to insert canonical property")

	// Setup: Seed property matched via alias
	_, err = pool.Exec(ctx, `
		INSERT INTO properties_core (
			listing_id, list_price, bedrooms_total, bathrooms_total, living_area, lot_size,
			year_built, days_on_market, standard_status, is_active, mlg_can_view,
			on_market_date, city, state, postal_code, street_number, street_name,
			latitude, longitude, partition_group,
			city_ref_id, city_alias_id, state_ref_id
		) VALUES (
			'PROP-ALIAS-002', 350000, 4, 3, 1800, 0.30,
			2018, 5, 'Active', true, true,
			$1, 'SA', 'TS', '12346', '200', 'Oak Ave',
			29.4242, -98.4937, 'active',
			$2, $3, $4
		)
	`, onMarket, canonicalCityID, aliasID, stateID)
	require.NoError(t, err, "failed to insert alias property")

	// Cleanup
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM properties_core WHERE listing_id IN ('PROP-CANONICAL-001', 'PROP-ALIAS-002')")
		_, _ = pool.Exec(ctx, "DELETE FROM city_aliases WHERE id = $1", aliasID)
		_, _ = pool.Exec(ctx, "DELETE FROM cities WHERE id = $1", canonicalCityID)
		_, _ = pool.Exec(ctx, "DELETE FROM counties WHERE id = $1", countyID)
		_, _ = pool.Exec(ctx, "DELETE FROM states WHERE id = $1", stateID)
	})

	// Execute: Search filtering by canonical city ID
	store := NewStore(pool, registry, "media.lzrcdn.com")
	req := SearchRequest{
		LocationFilters: LocationFilters{
			CityRefIDs: IDList{
				Values: []int64{canonicalCityID},
			},
		},
	}

	result, err := store.Search(ctx, req, 100)
	require.NoError(t, err, "search failed")

	// Assert: BOTH properties should appear in results
	require.Len(t, result.Results, 2, "Expected both canonical and alias-matched properties")

	// Verify both listing IDs are present
	listingIDs := make([]string, len(result.Results))
	for i, r := range result.Results {
		listingIDs[i] = r.ListingID
	}
	assert.Contains(t, listingIDs, "PROP-CANONICAL-001", "canonical property should be in results")
	assert.Contains(t, listingIDs, "PROP-ALIAS-002", "alias-matched property should be in results")

	// Verify alias metadata is preserved on alias-matched property
	var aliasProperty *ListingResult
	for i := range result.Results {
		if result.Results[i].ListingID == "PROP-ALIAS-002" {
			aliasProperty = &result.Results[i]
			break
		}
	}
	require.NotNil(t, aliasProperty, "alias property should be in results")
	require.NotNil(t, aliasProperty.City, "city field should not be nil")
	assert.Equal(t, "SA", *aliasProperty.City, "original raw city name should be preserved")
}

// TestSearchIncludesAliasMatchedSubdivisions verifies that when a user selects a canonical subdivision
// from autocomplete, the search results include ALL properties normalized to that subdivision,
// including properties that were originally listed with alternate names (aliases).
func TestSearchIncludesAliasMatchedSubdivisions(t *testing.T) {
	pool := testutil.NewPool(t)
	if pool == nil {
		t.Skip("test DB not configured")
		return
	}
	registry := db.NewRegistry()
	ctx := context.Background()

	// Setup: Create state
	var stateID int64
	err := pool.QueryRow(ctx, `
		INSERT INTO states (name, abbrev, normalized_name, slug, is_authoritative)
		VALUES ('Test State 2', 'T2', 'test state 2', 'test-state-2', true)
		RETURNING id
	`).Scan(&stateID)
	require.NoError(t, err, "failed to create test state")

	// Setup: Create county
	var countyID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO counties (name, normalized_name, slug, state_id, is_authoritative)
		VALUES ('Test County 2', 'test county 2', 'test-county-2', $1, true)
		RETURNING id
	`, stateID).Scan(&countyID)
	require.NoError(t, err, "failed to create test county")

	// Setup: Create city
	var cityID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO cities (name, normalized_name, slug, state_id, county_id, is_authoritative)
		VALUES ('Test City', 'test city', 'test-city', $1, $2, true)
		RETURNING id
	`, stateID, countyID).Scan(&cityID)
	require.NoError(t, err, "failed to create city")

	// Setup: Create canonical subdivision
	var canonicalSubdivisionID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO subdivisions (name, normalized_name, slug, state_id, primary_county_id, is_authoritative)
		VALUES ('Alamo Heights', 'alamo heights', 'alamo-heights', $1, $2, true)
		RETURNING id
	`, stateID, countyID).Scan(&canonicalSubdivisionID)
	require.NoError(t, err, "failed to create canonical subdivision")

	// Setup: Create matched alias
	var aliasID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO subdivision_aliases (raw_name, normalized_name, state_id, county_id, city_id, subdivision_id, status, confidence_score, match_count)
		VALUES ('Alamo Hts', 'alamo hts', $1, $2, $3, $4, 'matched', 0.90, 50)
		RETURNING id
	`, stateID, countyID, cityID, canonicalSubdivisionID).Scan(&aliasID)
	require.NoError(t, err, "failed to create subdivision alias")

	// Setup: Seed property matched via canonical name
	onMarket := time.Now().Add(-24 * time.Hour)
	_, err = pool.Exec(ctx, `
		INSERT INTO properties_core (
			listing_id, list_price, bedrooms_total, bathrooms_total, living_area, lot_size,
			year_built, days_on_market, standard_status, is_active, mlg_can_view,
			on_market_date, subdivision_name, city, state, postal_code, street_number, street_name,
			latitude, longitude, partition_group,
			subdivision_ref_id, city_ref_id, state_ref_id
		) VALUES (
			'PROP-SUBDIVISION-CANONICAL-001', 400000, 3, 2, 1600, 0.28,
			2016, 12, 'Active', true, true,
			$1, 'Alamo Heights', 'Test City', 'T2', '78209', '300', 'Broadway',
			29.4800, -98.4650, 'active',
			$2, $3, $4
		)
	`, onMarket, canonicalSubdivisionID, cityID, stateID)
	require.NoError(t, err, "failed to insert canonical subdivision property")

	// Setup: Seed property matched via alias
	_, err = pool.Exec(ctx, `
		INSERT INTO properties_core (
			listing_id, list_price, bedrooms_total, bathrooms_total, living_area, lot_size,
			year_built, days_on_market, standard_status, is_active, mlg_can_view,
			on_market_date, subdivision_name, city, state, postal_code, street_number, street_name,
			latitude, longitude, partition_group,
			subdivision_ref_id, subdivision_alias_id, city_ref_id, state_ref_id
		) VALUES (
			'PROP-SUBDIVISION-ALIAS-002', 450000, 4, 3, 1900, 0.32,
			2019, 7, 'Active', true, true,
			$1, 'Alamo Hts', 'Test City', 'T2', '78210', '400', 'Austin Hwy',
			29.4801, -98.4651, 'active',
			$2, $3, $4, $5
		)
	`, onMarket, canonicalSubdivisionID, aliasID, cityID, stateID)
	require.NoError(t, err, "failed to insert alias subdivision property")

	// Cleanup
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM properties_core WHERE listing_id IN ('PROP-SUBDIVISION-CANONICAL-001', 'PROP-SUBDIVISION-ALIAS-002')")
		_, _ = pool.Exec(ctx, "DELETE FROM subdivision_aliases WHERE id = $1", aliasID)
		_, _ = pool.Exec(ctx, "DELETE FROM subdivisions WHERE id = $1", canonicalSubdivisionID)
		_, _ = pool.Exec(ctx, "DELETE FROM cities WHERE id = $1", cityID)
		_, _ = pool.Exec(ctx, "DELETE FROM counties WHERE id = $1", countyID)
		_, _ = pool.Exec(ctx, "DELETE FROM states WHERE id = $1", stateID)
	})

	// Execute: Search filtering by canonical subdivision ID
	store := NewStore(pool, registry, "media.lzrcdn.com")
	req := SearchRequest{
		LocationFilters: LocationFilters{
			SubdivisionRefIDs: IDList{
				Values: []int64{canonicalSubdivisionID},
			},
		},
	}

	result, err := store.Search(ctx, req, 100)
	require.NoError(t, err, "search failed")

	// Assert: BOTH properties should appear in results
	require.Len(t, result.Results, 2, "Expected both canonical and alias-matched subdivision properties")

	// Verify both listing IDs are present
	listingIDs := make([]string, len(result.Results))
	for i, r := range result.Results {
		listingIDs[i] = r.ListingID
	}
	assert.Contains(t, listingIDs, "PROP-SUBDIVISION-CANONICAL-001", "canonical subdivision property should be in results")
	assert.Contains(t, listingIDs, "PROP-SUBDIVISION-ALIAS-002", "alias-matched subdivision property should be in results")

	// Verify alias metadata is preserved on alias-matched property
	var aliasProperty *ListingResult
	for i := range result.Results {
		if result.Results[i].ListingID == "PROP-SUBDIVISION-ALIAS-002" {
			aliasProperty = &result.Results[i]
			break
		}
	}
	require.NotNil(t, aliasProperty, "alias subdivision property should be in results")
	require.NotNil(t, aliasProperty.SubdivisionName, "subdivision name field should not be nil")
	assert.Equal(t, "Alamo Hts", *aliasProperty.SubdivisionName, "original raw subdivision name should be preserved")
}

// TestSearchActiveOnlyWithCityFilter reproduces the production 500 error:
// "argument of AND must be type boolean, not type integer (SQLSTATE 42804)"
// when searching with active_only=true and a city focus area filter.
func TestSearchActiveOnlyWithCityFilter(t *testing.T) {
	pool := testutil.NewPool(t)
	if pool == nil {
		t.Skip("test DB not configured")
		return
	}
	registry := db.NewRegistry()
	ctx := context.Background()

	// Setup: Create state
	var stateID int64
	err := pool.QueryRow(ctx, `
		INSERT INTO states (name, abbrev, normalized_name, slug, is_authoritative)
		VALUES ('Active City Test State', 'AC', 'active city test state', 'active-city-test-state', true)
		RETURNING id
	`).Scan(&stateID)
	require.NoError(t, err, "failed to create test state")

	// Setup: Create county
	var countyID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO counties (name, normalized_name, slug, state_id, is_authoritative)
		VALUES ('Active City Test County', 'active city test county', 'active-city-test-county', $1, true)
		RETURNING id
	`, stateID).Scan(&countyID)
	require.NoError(t, err, "failed to create test county")

	// Setup: Create city
	var cityID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO cities (name, normalized_name, slug, state_id, county_id, is_authoritative)
		VALUES ('Active City Test', 'active city test', 'active-city-test', $1, $2, true)
		RETURNING id
	`, stateID, countyID).Scan(&cityID)
	require.NoError(t, err, "failed to create city")

	// Setup: Seed active property
	onMarket := time.Now().Add(-24 * time.Hour)
	_, err = pool.Exec(ctx, `
		INSERT INTO properties_core (
			listing_id, list_price, bedrooms_total, bathrooms_total, living_area, lot_size,
			year_built, days_on_market, standard_status, is_active, mlg_can_view,
			on_market_date, city, state, postal_code, street_number, street_name,
			latitude, longitude, partition_group,
			city_ref_id, state_ref_id
		) VALUES (
			'PROP-ACTIVE-CITY-001', 250000, 3, 2, 1400, 0.20,
			2020, 5, 'Active', true, true,
			$1, 'Active City Test', 'AC', '99999', '500', 'Test Rd',
			30.0, -97.0, 'active',
			$2, $3
		)
	`, onMarket, cityID, stateID)
	require.NoError(t, err, "failed to insert property")

	// Cleanup
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM properties_core WHERE listing_id = 'PROP-ACTIVE-CITY-001'")
		_, _ = pool.Exec(ctx, "DELETE FROM cities WHERE id = $1", cityID)
		_, _ = pool.Exec(ctx, "DELETE FROM counties WHERE id = $1", countyID)
		_, _ = pool.Exec(ctx, "DELETE FROM states WHERE id = $1", stateID)
	})

	// Execute: Search with active_only=true + city filter (reproduces production request)
	store := NewStore(pool, registry, "media.lzrcdn.com")
	activeTrue := true
	req := SearchRequest{
		Params: SearchParams{
			ActiveOnly: BoolParam{Value: &activeTrue},
		},
		LocationFilters: LocationFilters{
			CityRefIDs: IDList{
				Values: []int64{cityID},
			},
		},
	}

	result, err := store.Search(ctx, req, 24)
	require.NoError(t, err, "search with active_only + city filter should not fail")

	assert.Equal(t, int64(1), result.Stats.TotalCount, "should find exactly 1 property")
	require.Len(t, result.Results, 1, "should return 1 result")
	assert.Equal(t, "PROP-ACTIVE-CITY-001", result.Results[0].ListingID)
}
