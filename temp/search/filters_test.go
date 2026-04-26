package search

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/db"
)

func TestExtractClause(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fragment string
		want     string
	}{
		{
			name:     "simple SELECT 1 with semicolon",
			fragment: "SELECT 1 FROM public.properties_core pc WHERE pc.is_active = true;",
			want:     "pc.is_active = true",
		},
		{
			name:     "simple SELECT 1 without semicolon",
			fragment: "SELECT 1 FROM public.properties_core pc WHERE pc.county_ref_id = ANY($2::bigint[])",
			want:     "pc.county_ref_id = ANY($2::bigint[])",
		},
		{
			name: "multi-line leading comments (city filter style)",
			fragment: `-- Filter properties by canonical city ID.
--
-- IMPORTANT: This query includes ALL properties normalized to this city,
-- regardless of which alias was matched during ingestion. The city_alias_id
-- field tracks which alternate name was matched, but search always filters
-- by the canonical city_ref_id to ensure complete results.
--
-- Example: Selecting "San Antonio" returns properties originally listed as:
--   - "San Antonio" (canonical)
--   - "SA" (alias)
--   - "San Ant" (alias)
--   - Any other approved aliases linked to San Antonio
SELECT 1 FROM public.properties_core pc WHERE pc.city_ref_id = ANY($3);`,
			want: "pc.city_ref_id = ANY($3)",
		},
		{
			name: "single leading comment (days-on-market style)",
			fragment: `-- min_days_on_market=N: on_market_date <= now() - N days
SELECT 1 FROM public.properties_core pc WHERE pc.on_market_date <= (now() - ($1 || ' days')::interval);`,
			want: "pc.on_market_date <= (now() - ($1 || ' days')::interval)",
		},
		{
			name: "subdivision filter with leading comments",
			fragment: `-- Filter properties by canonical subdivision ID.
--
-- IMPORTANT: This query includes ALL properties normalized to this subdivision,
-- regardless of which alias was matched during ingestion.
SELECT 1 FROM public.properties_core pc WHERE pc.subdivision_ref_id = ANY($4);`,
			want: "pc.subdivision_ref_id = ANY($4)",
		},
		{
			name: "EXISTS subquery (postal code filter)",
			fragment: `SELECT 1 FROM public.properties_core pc WHERE EXISTS (
    SELECT 1
    FROM public.postal_codes z
    WHERE z.id = ANY($1)
      AND z.geom IS NOT NULL
      AND pc.location IS NOT NULL
      AND ST_Contains(z.geom, pc.location::geometry)
);`,
			want: `EXISTS (
    SELECT 1
    FROM public.postal_codes z
    WHERE z.id = ANY($1)
      AND z.geom IS NOT NULL
      AND pc.location IS NOT NULL
      AND ST_Contains(z.geom, pc.location::geometry)
)`,
		},
		{
			name:     "non-SELECT 1 fragment returned as-is",
			fragment: "pc.list_price >= $5",
			want:     "pc.list_price >= $5",
		},
		{
			name:     "empty string",
			fragment: "",
			want:     "",
		},
		{
			name:     "semicolon only",
			fragment: ";",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractClause(tt.fragment)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestReplaceArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		fragment   string
		startIndex int
		argCount   int
		want       string
	}{
		{
			name:       "single arg reindex",
			fragment:   "pc.county_ref_id = ANY($1)",
			startIndex: 3,
			argCount:   1,
			want:       "pc.county_ref_id = ANY($3)",
		},
		{
			name:       "two args reindex",
			fragment:   "ST_DWithin(pc.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)",
			startIndex: 5,
			argCount:   3,
			want:       "ST_DWithin(pc.location, ST_SetSRID(ST_MakePoint($5, $6), 4326)::geography, $7)",
		},
		{
			name:       "no args",
			fragment:   "pc.is_active = true",
			startIndex: 1,
			argCount:   0,
			want:       "pc.is_active = true",
		},
		{
			name:       "arg in comments not affected when no $1 present in comments",
			fragment:   "-- some comment\nSELECT 1 FROM pc WHERE pc.city_ref_id = ANY($1)",
			startIndex: 4,
			argCount:   1,
			want:       "-- some comment\nSELECT 1 FROM pc WHERE pc.city_ref_id = ANY($4)",
		},
		{
			name:       "high startIndex multi-arg no corruption",
			fragment:   "ST_DWithin(pc.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)",
			startIndex: 11,
			argCount:   3,
			want:       "ST_DWithin(pc.location, ST_SetSRID(ST_MakePoint($11, $12), 4326)::geography, $13)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := replaceArgs(tt.fragment, tt.startIndex, tt.argCount)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestApplyFiltersActiveOnlyWithCity(t *testing.T) {
	t.Parallel()
	registry := db.NewRegistry()

	activeTrue := true
	req := SearchRequest{
		Params: SearchParams{
			ActiveOnly: BoolParam{Value: &activeTrue},
		},
		LocationFilters: LocationFilters{
			CityRefIDs: IDList{
				Values: []int64{851},
			},
		},
	}

	builder := newQueryBuilder(registry)
	applyFilters(builder, req)

	whereClause := builder.whereClause()
	args := builder.argsSlice()

	t.Logf("whereClause: %s", whereClause)
	t.Logf("args (%d): %+v", len(args), args)
	for i, a := range args {
		t.Logf("  arg[%d]: type=%T value=%v", i, a, a)
	}

	// Verify WHERE clause structure
	assert.Contains(t, whereClause, "pc.is_active = true")
	assert.Contains(t, whereClause, "pc.partition_group = $1")
	assert.Contains(t, whereClause, "pc.city_ref_id = ANY($2::bigint[])")

	// Verify args
	require.Len(t, args, 2)
	assert.Equal(t, "active", args[0])
	assert.Equal(t, []int64{851}, args[1])

	// Now inject into SearchStatsBase and verify full SQL
	statsSQL := registry.SQL(db.QueryName("SearchStatsBase"))
	injected := injectSearchClauses(statsSQL, whereClause, "")

	t.Logf("injected stats SQL:\n%s", injected)

	// The injected SQL should NOT contain raw integer literals as AND operands
	// Each AND operand should be a boolean expression
	whereIdx := strings.LastIndex(injected, "WHERE pc.mlg_can_view = true")
	require.Greater(t, whereIdx, 0, "should find WHERE clause in injected SQL")

	afterWhere := injected[whereIdx:]
	t.Logf("WHERE clause: %s", afterWhere)

	// Verify the WHERE clause has the expected structure
	assert.Contains(t, afterWhere, "AND pc.is_active = true")
	assert.Contains(t, afterWhere, "AND pc.partition_group = $1")
	assert.Contains(t, afterWhere, "AND (pc.city_ref_id = ANY($2::bigint[]))")
}

// Test SEO-focused boolean filters (Phase 11)
func TestApplyFiltersActiveAdultCommunity(t *testing.T) {
	t.Parallel()
	registry := db.NewRegistry()

	activeAdultTrue := true
	req := SearchRequest{
		Params: SearchParams{
			ActiveAdultCommunity: BoolParam{Value: &activeAdultTrue},
		},
	}

	builder := newQueryBuilder(registry)
	applyFilters(builder, req)

	whereClause := builder.whereClause()
	assert.Contains(t, whereClause, "pc.senior_community_yn = true")
}

func TestApplyFiltersAssociation(t *testing.T) {
	t.Parallel()
	registry := db.NewRegistry()

	associationTrue := true
	req := SearchRequest{
		Params: SearchParams{
			Association: BoolParam{Value: &associationTrue},
		},
	}

	builder := newQueryBuilder(registry)
	applyFilters(builder, req)

	whereClause := builder.whereClause()
	assert.Contains(t, whereClause, "pc.association_yn = true")
}

func TestApplyFiltersFireplace(t *testing.T) {
	t.Parallel()
	registry := db.NewRegistry()

	fireplaceTrue := true
	req := SearchRequest{
		Params: SearchParams{
			Fireplace: BoolParam{Value: &fireplaceTrue},
		},
	}

	builder := newQueryBuilder(registry)
	applyFilters(builder, req)

	whereClause := builder.whereClause()
	assert.Contains(t, whereClause, "pc.fireplace_yn = true")
}

func TestApplyFiltersSpa(t *testing.T) {
	t.Parallel()
	registry := db.NewRegistry()

	spaTrue := true
	req := SearchRequest{
		Params: SearchParams{
			Spa: BoolParam{Value: &spaTrue},
		},
	}

	builder := newQueryBuilder(registry)
	applyFilters(builder, req)

	whereClause := builder.whereClause()
	assert.Contains(t, whereClause, "pc.spa_yn = true")
}

func TestApplyFiltersForLease(t *testing.T) {
	t.Parallel()
	registry := db.NewRegistry()

	forLeaseTrue := true
	req := SearchRequest{
		Params: SearchParams{
			ForLease: BoolParam{Value: &forLeaseTrue},
		},
	}

	builder := newQueryBuilder(registry)
	applyFilters(builder, req)

	whereClause := builder.whereClause()
	assert.Contains(t, whereClause, "pc.mfr_for_lease_yn = true")
}

func TestApplyFiltersGarage(t *testing.T) {
	t.Parallel()
	registry := db.NewRegistry()

	garageTrue := true
	req := SearchRequest{
		Params: SearchParams{
			Garage: BoolParam{Value: &garageTrue},
		},
	}

	builder := newQueryBuilder(registry)
	applyFilters(builder, req)

	whereClause := builder.whereClause()
	assert.Contains(t, whereClause, "pc.garage_yn = true")
}

// Test range filters for SEO (Phase 11)
func TestApplyFiltersMinMonthlyFees(t *testing.T) {
	t.Parallel()
	registry := db.NewRegistry()

	minFees := 200.0
	req := SearchRequest{
		Params: SearchParams{
			MinMonthlyFees: FloatParam{Value: &minFees},
		},
	}

	builder := newQueryBuilder(registry)
	applyFilters(builder, req)

	whereClause := builder.whereClause()
	assert.Contains(t, whereClause, "pc.mfr_total_monthly_fees >=")
}

func TestApplyFiltersMaxMonthlyFees(t *testing.T) {
	t.Parallel()
	registry := db.NewRegistry()

	maxFees := 500.0
	req := SearchRequest{
		Params: SearchParams{
			MaxMonthlyFees: FloatParam{Value: &maxFees},
		},
	}

	builder := newQueryBuilder(registry)
	applyFilters(builder, req)

	whereClause := builder.whereClause()
	assert.Contains(t, whereClause, "pc.mfr_total_monthly_fees IS NULL OR pc.mfr_total_monthly_fees <=")
}

func TestApplyFiltersMinStories(t *testing.T) {
	t.Parallel()
	registry := db.NewRegistry()

	minStories := 2
	req := SearchRequest{
		Params: SearchParams{
			MinStories: IntParam{Value: &minStories},
		},
	}

	builder := newQueryBuilder(registry)
	applyFilters(builder, req)

	whereClause := builder.whereClause()
	assert.Contains(t, whereClause, "pc.stories_total >=")
}

func TestApplyFiltersMaxStories(t *testing.T) {
	t.Parallel()
	registry := db.NewRegistry()

	maxStories := 1
	req := SearchRequest{
		Params: SearchParams{
			MaxStories: IntParam{Value: &maxStories},
		},
	}

	builder := newQueryBuilder(registry)
	applyFilters(builder, req)

	whereClause := builder.whereClause()
	assert.Contains(t, whereClause, "pc.stories_total <=")
}

// Test combined SEO filters
func TestApplyFiltersCombinedSEO(t *testing.T) {
	t.Parallel()
	registry := db.NewRegistry()

	activeAdultTrue := true
	fireplaceTrue := true
	maxFees := 400.0

	req := SearchRequest{
		Params: SearchParams{
			ActiveAdultCommunity: BoolParam{Value: &activeAdultTrue},
			Fireplace:            BoolParam{Value: &fireplaceTrue},
			MaxMonthlyFees:       FloatParam{Value: &maxFees},
		},
		LocationFilters: LocationFilters{
			CityRefIDs: IDList{Values: []int64{101}},
		},
	}

	builder := newQueryBuilder(registry)
	applyFilters(builder, req)

	whereClause := builder.whereClause()

	// Should contain all three filters
	assert.Contains(t, whereClause, "pc.senior_community_yn = true")
	assert.Contains(t, whereClause, "pc.fireplace_yn = true")
	assert.Contains(t, whereClause, "pc.mfr_total_monthly_fees IS NULL OR pc.mfr_total_monthly_fees <=")
	assert.Contains(t, whereClause, "pc.city_ref_id = ANY")
}
