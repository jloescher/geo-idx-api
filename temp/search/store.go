package search

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xotec-solutions/xotec-datalayer/src/internal/db"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/mls"
)

var nyLoc = func() *time.Location {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Printf("[search] warning: failed to load America/New_York timezone, falling back to UTC: %v", err)
		return time.UTC
	}
	return loc
}()

// Store executes search queries.
type Store struct {
	Pool         *pgxpool.Pool
	Registry     *db.Registry
	MediaCDNHost string
}

func NewStore(pool *pgxpool.Pool, registry *db.Registry, mediaCDNHost string) *Store {
	return &Store{Pool: pool, Registry: registry, MediaCDNHost: mediaCDNHost}
}

// SearchStats holds aggregate stats for a search.
type SearchStats struct {
	TotalCount  int64   `json:"total_count"`
	AvgDOM      float64 `json:"avg_dom"`
	AvgPrice    float64 `json:"avg_price"`
	AvgPPSF     float64 `json:"avg_ppsf"`
	MedianPrice float64 `json:"median_price"`
}

// PrimaryImage holds media details for the primary image.
type PrimaryImage struct {
	MediaKey    *string              `json:"media_key"`
	Sources     *PrimaryImageSources `json:"sources,omitempty"`
	IsOptimized *bool                `json:"is_optimized"`
}

type PrimaryImageSources struct {
	AVIF *string `json:"avif"`
	WebP *string `json:"webp"`
}

// ListingResult is returned for each listing card.
type ListingResult struct {
	ID              int64   `json:"id"`
	ListingID       string  `json:"listing_id"`
	StandardStatus  *string `json:"standard_status,omitempty"`
	PropertyType    *string `json:"property_type,omitempty"`
	PropertySubType *string `json:"property_sub_type,omitempty"`

	// Address fields
	StreetNumber    *string `json:"street_number,omitempty"`
	StreetDirPrefix *string `json:"street_dir_prefix,omitempty"`
	StreetName      *string `json:"street_name,omitempty"`
	StreetSuffix    *string `json:"street_suffix,omitempty"`
	StreetDirSuffix *string `json:"street_dir_suffix,omitempty"`
	UnitNumber      *string `json:"unit_number,omitempty"`
	City            *string `json:"city,omitempty"`
	CountyOrParish  *string `json:"county_or_parish,omitempty"`
	State           *string `json:"state_or_province,omitempty"`
	PostalCode      *string `json:"postal_code,omitempty"`

	SubdivisionName    *string  `json:"subdivision_name,omitempty"`
	ArchitecturalStyle []string `json:"architectural_style,omitempty"`

	// Ref IDs
	CityRefID        *int64 `json:"city_ref_id,omitempty"`
	CountyRefID      *int64 `json:"county_ref_id,omitempty"`
	StateRefID       *int64 `json:"state_ref_id,omitempty"`
	PostalCodeRefID  *int64 `json:"postal_code_ref_id,omitempty"`
	SubdivisionRefID *int64 `json:"subdivision_ref_id,omitempty"`

	// Pricing
	ListPrice         *float64 `json:"list_price,omitempty"`
	PreviousListPrice *float64 `json:"previous_list_price,omitempty"`
	OriginalListPrice *float64 `json:"original_list_price,omitempty"`
	ClosePrice        *float64 `json:"close_price,omitempty"`

	// Property details
	BedroomsTotal       *int     `json:"bedrooms_total,omitempty"`
	BathroomsTotal      *int     `json:"bathrooms_total,omitempty"`
	BathroomsFull       *int     `json:"bathrooms_full,omitempty"`
	BathroomsHalf       *int     `json:"bathrooms_half,omitempty"`
	LivingArea          *int     `json:"living_area,omitempty"`
	LotSizeAcres        *float64 `json:"lot_size_acres,omitempty"`
	YearBuilt           *int     `json:"year_built,omitempty"`
	StoriesTotal        *int     `json:"stories_total,omitempty"`
	MfrFloorNumber      *int     `json:"mfr_floor_number,omitempty"`
	GarageSpaces        *int     `json:"garage_spaces,omitempty"`
	GarageYn            *bool    `json:"garage_yn,omitempty"`
	MfrTotalMonthlyFees *float64 `json:"mfr_total_monthly_fees,omitempty"`
	AssociationYn       *bool    `json:"association_yn,omitempty"`

	// Status
	MLSStatus         *string    `json:"mls_status,omitempty"`
	IsActive          *bool      `json:"is_active,omitempty"`
	IsCurrentlyActive *bool      `json:"is_currently_active,omitempty"`
	BecameInactiveAt  *time.Time `json:"became_inactive_at,omitempty"`
	LastActivityAt    *time.Time `json:"last_activity_at,omitempty"`
	OnMarketDate      *time.Time `json:"on_market_date,omitempty"`
	DaysOnMarket      *int       `json:"days_on_market,omitempty"`
	CloseDate         *time.Time `json:"close_date,omitempty"`

	PhotosCount *int `json:"photos_count,omitempty"`

	// Agent/office MLS IDs
	ListAgentMlsID     *string `json:"list_agent_mls_id,omitempty"`
	CoListAgentMlsID   *string `json:"co_list_agent_mls_id,omitempty"`
	ListOfficeMlsID    *string `json:"list_office_mls_id,omitempty"`
	CoListOfficeMlsID  *string `json:"co_list_office_mls_id,omitempty"`
	BuyerAgentMlsID    *string `json:"buyer_agent_mls_id,omitempty"`
	CoBuyerAgentMlsID  *string `json:"co_buyer_agent_mls_id,omitempty"`
	BuyerOfficeMlsID   *string `json:"buyer_office_mls_id,omitempty"`
	CoBuyerOfficeMlsID *string `json:"co_buyer_office_mls_id,omitempty"`

	// Boolean feature flags
	MfrWaterViewYn     *bool `json:"mfr_water_view_yn,omitempty"`
	PoolPrivateYn      *bool `json:"pool_private_yn,omitempty"`
	SeniorCommunityYn  *bool `json:"senior_community_yn,omitempty"`
	WaterfrontYn       *bool `json:"waterfront_yn,omitempty"`
	MfrDockYn          *bool `json:"mfr_dock_yn,omitempty"`
	NewConstructionYn  *bool `json:"new_construction_yn,omitempty"`
	LowRiskFloodzoneYn *bool `json:"low_risk_floodzone_yn,omitempty"`

	// Coordinates
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`

	Slug *string `json:"slug,omitempty"`

	// School ref IDs
	ElementarySchoolRefID *int64 `json:"elementary_school_ref_id,omitempty"`
	MiddleSchoolRefID     *int64 `json:"middle_school_ref_id,omitempty"`
	HighSchoolRefID       *int64 `json:"high_school_ref_id,omitempty"`

	// Alias IDs
	CityAliasID        *int64 `json:"city_alias_id,omitempty"`
	SubdivisionAliasID *int64 `json:"subdivision_alias_id,omitempty"`

	// Timestamps
	PriceChangeTimestamp  *time.Time `json:"price_change_timestamp,omitempty"`
	PhotosChangeTimestamp *time.Time `json:"photos_change_timestamp,omitempty"`
	ModificationTimestamp *time.Time `json:"modification_timestamp,omitempty"`
	CreatedAt             *time.Time `json:"created_at,omitempty"`
	UpdatedAt             *time.Time `json:"updated_at,omitempty"`

	// Internal field — not serialized to JSON. Used by handler for RTO computation.
	TaxAnnualAmount *float64 `json:"-"`

	// RTO estimate — only populated when include_rto is requested.
	RTOEstimate *RTOEstimate `json:"rto_estimate,omitempty"`

	// Computed fields
	DistanceMiles *float64       `json:"distance_miles,omitempty"`
	Address       ListingAddress `json:"address"`
	PrimaryImage  PrimaryImage   `json:"primary_image"`
}

// RTOEstimate holds the recommended rent-to-own estimate for a listing.
type RTOEstimate struct {
	EstimatedMonthly       float64 `json:"estimated_monthly"`
	EstimatedPurchasePrice float64 `json:"estimated_purchase_price"`
}

// SearchResult holds the search response components.
type SearchResult struct {
	Results    []ListingResult
	Stats      SearchStats
	HasMore    bool
	NextCursor *string
}

type searchRow struct {
	// properties_core columns (matches PropertiesCore model)
	ID                    int64              `db:"id"`
	ListingID             pgtype.Text        `db:"listing_id"`
	StandardStatus        pgtype.Text        `db:"standard_status"`
	PartitionGroup        pgtype.Text        `db:"partition_group"`
	PropertyType          pgtype.Text        `db:"property_type"`
	PropertySubType       pgtype.Text        `db:"property_sub_type"`
	ArchitecturalStyle    []string           `db:"architectural_style"`
	StreetNumber          pgtype.Text        `db:"street_number"`
	StreetDirPrefix       pgtype.Text        `db:"street_dir_prefix"`
	StreetName            pgtype.Text        `db:"street_name"`
	StreetSuffix          pgtype.Text        `db:"street_suffix"`
	StreetDirSuffix       pgtype.Text        `db:"street_dir_suffix"`
	UnitNumber            pgtype.Text        `db:"unit_number"`
	City                  pgtype.Text        `db:"city"`
	CountyOrParish        pgtype.Text        `db:"county_or_parish"`
	State                 pgtype.Text        `db:"state_or_province"`
	PostalCode            pgtype.Text        `db:"postal_code"`
	SubdivisionName       pgtype.Text        `db:"subdivision_name"`
	CityRefID             pgtype.Int8        `db:"city_ref_id"`
	CountyRefID           pgtype.Int8        `db:"county_ref_id"`
	StateRefID            pgtype.Int8        `db:"state_ref_id"`
	PostalCodeRefID       pgtype.Int8        `db:"postal_code_ref_id"`
	SubdivisionRefID      pgtype.Int8        `db:"subdivision_ref_id"`
	ListPrice             pgtype.Numeric     `db:"list_price"`
	PreviousListPrice     pgtype.Numeric     `db:"previous_list_price"`
	OriginalListPrice     pgtype.Numeric     `db:"original_list_price"`
	ClosePrice            pgtype.Numeric     `db:"close_price"`
	BedroomsTotal         pgtype.Int4        `db:"bedrooms_total"`
	BathroomsTotal        pgtype.Int4        `db:"bathrooms_total"`
	BathroomsFull         pgtype.Int4        `db:"bathrooms_full"`
	BathroomsHalf         pgtype.Int4        `db:"bathrooms_half"`
	LivingArea            pgtype.Int4        `db:"living_area"`
	LotSizeAcres          pgtype.Numeric     `db:"lot_size_acres"`
	YearBuilt             pgtype.Int4        `db:"year_built"`
	StoriesTotal          pgtype.Int4        `db:"stories_total"`
	MfrFloorNumber        pgtype.Int4        `db:"mfr_floor_number"`
	GarageSpaces          pgtype.Int4        `db:"garage_spaces"`
	GarageYn              pgtype.Bool        `db:"garage_yn"`
	MfrTotalMonthlyFees   pgtype.Numeric     `db:"mfr_total_monthly_fees"`
	AssociationYn         pgtype.Bool        `db:"association_yn"`
	MLSStatus             pgtype.Text        `db:"mls_status"`
	IsActive              pgtype.Bool        `db:"is_active"`
	IsCurrentlyActive     pgtype.Bool        `db:"is_currently_active"`
	BecameInactiveAt      pgtype.Timestamptz `db:"became_inactive_at"`
	LastActivityAt        pgtype.Timestamptz `db:"last_activity_at"`
	PhotosCount           pgtype.Int4        `db:"photos_count"`
	MlgCanView            pgtype.Bool        `db:"mlg_can_view"`
	ListAgentMlsID        pgtype.Text        `db:"list_agent_mls_id"`
	CoListAgentMlsID      pgtype.Text        `db:"co_list_agent_mls_id"`
	ListOfficeMlsID       pgtype.Text        `db:"list_office_mls_id"`
	CoListOfficeMlsID     pgtype.Text        `db:"co_list_office_mls_id"`
	BuyerAgentMlsID       pgtype.Text        `db:"buyer_agent_mls_id"`
	CoBuyerAgentMlsID     pgtype.Text        `db:"co_buyer_agent_mls_id"`
	BuyerOfficeMlsID      pgtype.Text        `db:"buyer_office_mls_id"`
	CoBuyerOfficeMlsID    pgtype.Text        `db:"co_buyer_office_mls_id"`
	MfrWaterViewYn        pgtype.Bool        `db:"mfr_water_view_yn"`
	PoolPrivateYn         pgtype.Bool        `db:"pool_private_yn"`
	SeniorCommunityYn     pgtype.Bool        `db:"senior_community_yn"`
	WaterfrontYn          pgtype.Bool        `db:"waterfront_yn"`
	MfrDockYn             pgtype.Bool        `db:"mfr_dock_yn"`
	NewConstructionYn     pgtype.Bool        `db:"new_construction_yn"`
	LowRiskFloodzoneYn    pgtype.Bool        `db:"low_risk_floodzone_yn"`
	Latitude              pgtype.Numeric     `db:"latitude"`
	Longitude             pgtype.Numeric     `db:"longitude"`
	Location              []byte             `db:"location"` // PostGIS geography; scanned as raw bytes, not exposed in JSON
	Slug                  pgtype.Text        `db:"slug"`
	ElementarySchoolRefID pgtype.Int8        `db:"elementary_school_ref_id"`
	MiddleSchoolRefID     pgtype.Int8        `db:"middle_school_ref_id"`
	HighSchoolRefID       pgtype.Int8        `db:"high_school_ref_id"`
	CityAliasID           pgtype.Int8        `db:"city_alias_id"`
	SubdivisionAliasID    pgtype.Int8        `db:"subdivision_alias_id"`
	DaysOnMarket          pgtype.Int4        `db:"days_on_market"` // may still exist in production
	OnMarketDate          pgtype.Timestamptz `db:"on_market_date"`
	CloseDate             pgtype.Timestamptz `db:"close_date"`
	PriceChangeTimestamp  pgtype.Timestamptz `db:"price_change_timestamp"`
	PhotosChangeTimestamp pgtype.Timestamptz `db:"photos_change_timestamp"`
	ModificationTimestamp pgtype.Timestamptz `db:"modification_timestamp"`
	CreatedAt             pgtype.Timestamptz `db:"created_at"`
	UpdatedAt             pgtype.Timestamptz `db:"updated_at"`

	// Computed columns from query
	DistanceMeters        pgtype.Float8  `db:"distance_meters"`
	PrimaryMediaKey       pgtype.Text    `db:"primary_media_key"`
	PrimaryHostedKey      pgtype.Text    `db:"primary_hosted_key"`
	PrimaryMediaOptimized pgtype.Bool    `db:"primary_media_optimized"`
	TaxAnnualAmount       pgtype.Numeric `db:"tax_annual_amount"`
}

type statsRow struct {
	TotalCount  int64          `db:"total_count"`
	AvgDOM      pgtype.Numeric `db:"avg_dom"`
	AvgPrice    pgtype.Numeric `db:"avg_price"`
	AvgPPSF     pgtype.Numeric `db:"avg_ppsf"`
	MedianPrice pgtype.Numeric `db:"median_price"`
}

// Search executes the search query and returns results and stats.
func (s *Store) Search(ctx context.Context, req SearchRequest, effectiveLimit int) (SearchResult, error) {
	distanceArgs, distanceAvailable := distanceSortArgs(req)
	sortSpec, sortDir := resolveSort(req, distanceAvailable)

	builder := newQueryBuilder(s.Registry)
	if sortSpec.isDistance {
		builder.argIndex = 3
	}

	applyFilters(builder, req)
	whereClause := builder.whereClause()
	whereArgs := append([]any{}, builder.argsSlice()...)
	argIndex := builder.argIndex

	cursorClause, cursorArgs, err := buildCursorClause(req, sortSpec, sortDir, argIndex, s.Registry)
	if err != nil {
		return SearchResult{}, err
	}
	if strings.TrimSpace(cursorClause) != "" {
		cursorClause = "AND " + cursorClause
	}

	queryArgs := make([]any, 0, len(distanceArgs)+len(whereArgs)+len(cursorArgs)+1)
	if sortSpec.isDistance {
		queryArgs = append(queryArgs, distanceArgs...)
	}
	queryArgs = append(queryArgs, whereArgs...)
	argIndex += len(cursorArgs)
	queryArgs = append(queryArgs, cursorArgs...)

	query := s.Registry.SQL(db.QueryName(sortSpec.selectQuery(sortDir)))
	query = injectSearchClauses(query, whereClause, cursorClause)
	query = strings.TrimSuffix(strings.TrimSpace(query), ";")
	query = fmt.Sprintf("%s\nLIMIT $%d", query, argIndex)
	queryArgs = append(queryArgs, effectiveLimit+1)

	// Build stats filter args independently (argIndex=1) so placeholders
	// are not offset by the distance sort args ($1/$2).
	statsBuilder := newQueryBuilder(s.Registry)
	applyFilters(statsBuilder, req)
	statsWhere := statsBuilder.whereClause()
	statsArgs := append([]any{}, statsBuilder.argsSlice()...)

	g, gCtx := errgroup.WithContext(ctx)

	var rows []searchRow
	g.Go(func() error {
		if err := pgxscan.Select(gCtx, s.Pool, &rows, query, queryArgs...); err != nil {
			return fmt.Errorf("search query: %w", err)
		}
		return nil
	})

	var stats SearchStats
	g.Go(func() error {
		var err error
		stats, err = s.fetchStats(gCtx, statsWhere, statsArgs)
		return err
	})

	if err := g.Wait(); err != nil {
		return SearchResult{}, err
	}

	hasMore := false
	if len(rows) > effectiveLimit {
		hasMore = true
		rows = rows[:effectiveLimit]
	}

	results := make([]ListingResult, 0, len(rows))
	var nextCursor *string
	for idx, row := range rows {
		result := mapRowToResult(row, s.MediaCDNHost)
		results = append(results, result)
		if idx == len(rows)-1 && hasMore {
			cursor, err := buildNextCursor(row, sortSpec)
			if err != nil {
				return SearchResult{}, err
			}
			nextCursor = &cursor
		}
	}

	return SearchResult{
		Results:    results,
		Stats:      stats,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}

func (s *Store) fetchStats(ctx context.Context, whereClause string, args []any) (SearchStats, error) {
	statsQuery := s.Registry.SQL(db.QueryName("SearchStatsBase"))
	statsQuery = injectSearchClauses(statsQuery, whereClause, "")
	var row statsRow
	if err := pgxscan.Get(ctx, s.Pool, &row, statsQuery, args...); err != nil {
		return SearchStats{}, fmt.Errorf("search stats: %w", err)
	}
	return SearchStats{
		TotalCount:  row.TotalCount,
		AvgDOM:      numericToFloat64(row.AvgDOM),
		AvgPrice:    numericToFloat64(row.AvgPrice),
		AvgPPSF:     numericToFloat64(row.AvgPPSF),
		MedianPrice: numericToFloat64(row.MedianPrice),
	}, nil
}

// Count returns the number of properties matching the search request.
func (s *Store) Count(ctx context.Context, req SearchRequest) (int64, error) {
	builder := newQueryBuilder(s.Registry)

	applyFilters(builder, req)
	whereClause := builder.whereClause()
	whereArgs := append([]any{}, builder.argsSlice()...)

	// Reuse the stats query, or a simplified count-only query
	// SearchStatsBase calculates count(*) as total_count, which is exactly what we need.
	// We can ignore the other aggregate fields for now.
	statsQuery := s.Registry.SQL(db.QueryName("SearchStatsBase"))
	statsQuery = injectSearchClauses(statsQuery, whereClause, "")

	var row statsRow
	// We use QueryRow because SearchStatsBase returns a single row of aggregates
	if err := pgxscan.Get(ctx, s.Pool, &row, statsQuery, whereArgs...); err != nil {
		return 0, fmt.Errorf("search count: %w", err)
	}

	return row.TotalCount, nil
}

func mapRowToResult(row searchRow, mediaCDNHost string) ListingResult {
	address := buildListingAddress(addressParts{
		StreetNumber: textValue(row.StreetNumber),
		StreetDirPre: textValue(row.StreetDirPrefix),
		StreetName:   textValue(row.StreetName),
		StreetSuffix: textValue(row.StreetSuffix),
		StreetDirSuf: textValue(row.StreetDirSuffix),
		UnitNumber:   textValue(row.UnitNumber),
		City:         textValue(row.City),
		State:        textValue(row.State),
		PostalCode:   textValue(row.PostalCode),
	})

	var distanceMiles *float64
	if row.DistanceMeters.Valid {
		miles := row.DistanceMeters.Float64 / 1609.344
		distanceMiles = &miles
	}

	var daysOnMarket *int
	if row.OnMarketDate.Valid {
		now := time.Now().In(nyLoc)
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, nyLoc)
		omd := row.OnMarketDate.Time.In(nyLoc)
		omdDay := time.Date(omd.Year(), omd.Month(), omd.Day(), 0, 0, 0, 0, nyLoc)
		dom := int(today.Sub(omdDay).Hours() / 24)
		if dom < 0 {
			dom = 0
		}
		daysOnMarket = &dom
	}

	rawListingID := textValue(row.ListingID)
	mediaKey := resolvePrimaryMediaKey(rawListingID, row.PrimaryMediaKey, row.PrimaryHostedKey)
	mediaSources := buildPrimaryImageSources(mediaCDNHost, rawListingID, mediaKey)

	return ListingResult{
		ID:              row.ID,
		ListingID:       mls.StripPrefix(rawListingID),
		StandardStatus:  textPtr(row.StandardStatus),
		PropertyType:    textPtr(row.PropertyType),
		PropertySubType: textPtr(row.PropertySubType),

		StreetNumber:    textPtr(row.StreetNumber),
		StreetDirPrefix: textPtr(row.StreetDirPrefix),
		StreetName:      textPtr(row.StreetName),
		StreetSuffix:    textPtr(row.StreetSuffix),
		StreetDirSuffix: textPtr(row.StreetDirSuffix),
		UnitNumber:      textPtr(row.UnitNumber),
		City:            textPtr(row.City),
		CountyOrParish:  textPtr(row.CountyOrParish),
		State:           textPtr(row.State),
		PostalCode:      textPtr(row.PostalCode),

		SubdivisionName:    textPtr(row.SubdivisionName),
		ArchitecturalStyle: stringSlice(row.ArchitecturalStyle),

		CityRefID:        int8Ptr(row.CityRefID),
		CountyRefID:      int8Ptr(row.CountyRefID),
		StateRefID:       int8Ptr(row.StateRefID),
		PostalCodeRefID:  int8Ptr(row.PostalCodeRefID),
		SubdivisionRefID: int8Ptr(row.SubdivisionRefID),

		ListPrice:         numericPtr(row.ListPrice),
		PreviousListPrice: numericPtr(row.PreviousListPrice),
		OriginalListPrice: numericPtr(row.OriginalListPrice),
		ClosePrice:        numericPtr(row.ClosePrice),

		BedroomsTotal:       intPtr(row.BedroomsTotal),
		BathroomsTotal:      intPtr(row.BathroomsTotal),
		BathroomsFull:       intPtr(row.BathroomsFull),
		BathroomsHalf:       intPtr(row.BathroomsHalf),
		LivingArea:          intPtr(row.LivingArea),
		LotSizeAcres:        numericPtr(row.LotSizeAcres),
		YearBuilt:           intPtr(row.YearBuilt),
		StoriesTotal:        intPtr(row.StoriesTotal),
		MfrFloorNumber:      intPtr(row.MfrFloorNumber),
		GarageSpaces:        intPtr(row.GarageSpaces),
		GarageYn:            boolPtr(row.GarageYn),
		MfrTotalMonthlyFees: numericPtr(row.MfrTotalMonthlyFees),
		AssociationYn:       boolPtr(row.AssociationYn),

		MLSStatus:         textPtr(row.MLSStatus),
		IsActive:          boolPtr(row.IsActive),
		IsCurrentlyActive: boolPtr(row.IsCurrentlyActive),
		BecameInactiveAt:  timePtr(row.BecameInactiveAt),
		LastActivityAt:    timePtr(row.LastActivityAt),
		OnMarketDate:      timePtr(row.OnMarketDate),
		DaysOnMarket:      daysOnMarket,
		CloseDate:         timePtr(row.CloseDate),

		PhotosCount: intPtr(row.PhotosCount),

		ListAgentMlsID:     textPtr(row.ListAgentMlsID),
		CoListAgentMlsID:   textPtr(row.CoListAgentMlsID),
		ListOfficeMlsID:    textPtr(row.ListOfficeMlsID),
		CoListOfficeMlsID:  textPtr(row.CoListOfficeMlsID),
		BuyerAgentMlsID:    textPtr(row.BuyerAgentMlsID),
		CoBuyerAgentMlsID:  textPtr(row.CoBuyerAgentMlsID),
		BuyerOfficeMlsID:   textPtr(row.BuyerOfficeMlsID),
		CoBuyerOfficeMlsID: textPtr(row.CoBuyerOfficeMlsID),

		MfrWaterViewYn:     boolPtr(row.MfrWaterViewYn),
		PoolPrivateYn:      boolPtr(row.PoolPrivateYn),
		SeniorCommunityYn:  boolPtr(row.SeniorCommunityYn),
		WaterfrontYn:       boolPtr(row.WaterfrontYn),
		MfrDockYn:          boolPtr(row.MfrDockYn),
		NewConstructionYn:  boolPtr(row.NewConstructionYn),
		LowRiskFloodzoneYn: boolPtr(row.LowRiskFloodzoneYn),

		Latitude:  numericPtr(row.Latitude),
		Longitude: numericPtr(row.Longitude),

		Slug: textPtr(row.Slug),

		ElementarySchoolRefID: int8Ptr(row.ElementarySchoolRefID),
		MiddleSchoolRefID:     int8Ptr(row.MiddleSchoolRefID),
		HighSchoolRefID:       int8Ptr(row.HighSchoolRefID),

		CityAliasID:        int8Ptr(row.CityAliasID),
		SubdivisionAliasID: int8Ptr(row.SubdivisionAliasID),

		PriceChangeTimestamp:  timePtr(row.PriceChangeTimestamp),
		PhotosChangeTimestamp: timePtr(row.PhotosChangeTimestamp),
		ModificationTimestamp: timePtr(row.ModificationTimestamp),
		CreatedAt:             timePtr(row.CreatedAt),
		UpdatedAt:             timePtr(row.UpdatedAt),

		TaxAnnualAmount: numericPtr(row.TaxAnnualAmount),

		DistanceMiles: distanceMiles,
		Address:       address,
		PrimaryImage: PrimaryImage{
			MediaKey:    stringPtr(mediaKey),
			Sources:     mediaSources,
			IsOptimized: boolPtr(row.PrimaryMediaOptimized),
		},
	}
}

func injectSearchClauses(query string, whereClause string, cursorClause string) string {
	result := strings.TrimSpace(query)
	if strings.TrimSpace(whereClause) != "" {
		result = strings.Replace(result, "WHERE pc.mlg_can_view = true", "WHERE pc.mlg_can_view = true "+whereClause, 1)
	}
	if strings.TrimSpace(cursorClause) != "" {
		result = strings.Replace(result, "WHERE 1=1", "WHERE 1=1 "+cursorClause, 1)
	}
	return strings.TrimSpace(result)
}

func distanceSortArgs(req SearchRequest) ([]any, bool) {
	if req.Params.Geo == nil {
		return nil, false
	}
	lat, lng, ok := req.Params.Geo.DistanceCenter()
	if !ok {
		return nil, false
	}
	return []any{lng, lat}, true
}

func replaceDistanceCursorArgs(fragment string, startIndex int) string {
	result := fragment
	result = strings.ReplaceAll(result, "$4", fmt.Sprintf("$%d", startIndex+1))
	result = strings.ReplaceAll(result, "$3", fmt.Sprintf("$%d", startIndex))
	return result
}

func resolvePrimaryMediaKey(listingID string, mediaKey pgtype.Text, hostedKey pgtype.Text) string {
	if mediaKey.Valid {
		value := strings.TrimSpace(mediaKey.String)
		if value != "" {
			return value
		}
	}
	if hostedKey.Valid {
		value := strings.TrimSpace(hostedKey.String)
		if value == "" {
			return ""
		}
		value = stripMediaExtension(value)
		if strings.Contains(value, "/") {
			parts := strings.Split(value, "/")
			value = parts[len(parts)-1]
		}
		if value == "" {
			return ""
		}
		return value
	}
	return ""
}

func buildPrimaryImageSources(cdnHost string, listingID string, mediaKey string) *PrimaryImageSources {
	if strings.TrimSpace(listingID) == "" || strings.TrimSpace(mediaKey) == "" {
		return nil
	}
	host := strings.TrimSpace(cdnHost)
	if host == "" {
		host = "media.lzrcdn.com"
	}
	avif := fmt.Sprintf("https://%s/%s/%s.avif", host, listingID, mediaKey)
	webp := fmt.Sprintf("https://%s/%s/%s.webp", host, listingID, mediaKey)
	return &PrimaryImageSources{AVIF: &avif, WebP: &webp}
}

func stripMediaExtension(value string) string {
	for _, ext := range []string{".avif", ".webp", ".jpg", ".jpeg", ".png", ".gif", ".bin"} {
		if strings.HasSuffix(strings.ToLower(value), ext) {
			return strings.TrimSuffix(value, ext)
		}
	}
	return value
}

func stringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func buildNextCursor(row searchRow, spec sortSpec) (string, error) {
	sortKey, err := extractSortKey(row, spec)
	if err != nil {
		return "", err
	}
	return EncodeCursor(sortKey, row.ID)
}

func numericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	val, err := n.Float64Value()
	if err != nil {
		return 0
	}
	return val.Float64
}

func numericPtr(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	val, err := n.Float64Value()
	if err != nil {
		return nil
	}
	return &val.Float64
}

func intPtr(n pgtype.Int4) *int {
	if !n.Valid {
		return nil
	}
	val := int(n.Int32)
	return &val
}

func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	val := t.String
	return &val
}

func textValue(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func boolPtr(b pgtype.Bool) *bool {
	if !b.Valid {
		return nil
	}
	val := b.Bool
	return &val
}

func timePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	val := t.Time
	return &val
}

func int8Ptr(n pgtype.Int8) *int64 {
	if !n.Valid {
		return nil
	}
	val := n.Int64
	return &val
}

func stringSlice(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return s
}

func applyFilters(builder *queryBuilder, req SearchRequest) {
	// Only apply ActiveOnly filter when explicitly set to true
	// If not present in request, don't apply the filter
	if req.Params.ActiveOnly.Value != nil && *req.Params.ActiveOnly.Value {
		builder.addClause("SearchFilterActiveOnly")
		// Partition Optimization: ActiveOnly targets the 'active' partition
		builder.addClause("SearchFilterPartitionGroup", "active")
	}

	if len(req.ListingTypes) > 0 {
		builder.addClause("SearchFilterListingTypes", req.ListingTypes)
	}

	if len(req.Params.PropertySubType) > 0 {
		builder.addClause("SearchFilterPropertySubTypes", req.Params.PropertySubType)
	}

	if len(req.Params.SpecialListingConditions) > 0 {
		builder.addClause("SearchFilterSpecialListingConditions", req.Params.SpecialListingConditions)
	}

	if len(req.Params.Statuses) > 0 {
		builder.addClause("SearchFilterStatuses", req.Params.Statuses)
		// Partition Optimization
		if group := detectPartitionGroup(req.Params.Statuses); group != "" {
			builder.addClause("SearchFilterPartitionGroup", group)
		}
	}

	if val, ok := req.Params.MinPrice.Float64(); ok {
		builder.addClause("SearchFilterListPriceMin", val)
	}
	if val, ok := req.Params.MaxPrice.Float64(); ok {
		builder.addClause("SearchFilterListPriceMax", val)
	}

	if val, ok := req.Params.MinBeds.Int(); ok {
		builder.addClause("SearchFilterBedroomsMin", val)
	}
	if val, ok := req.Params.MaxBeds.Int(); ok {
		builder.addClause("SearchFilterBedroomsMax", val)
	}

	if val, ok := req.Params.MinBaths.Int(); ok {
		builder.addClause("SearchFilterBathroomsMin", val)
	}
	if val, ok := req.Params.MaxBaths.Int(); ok {
		builder.addClause("SearchFilterBathroomsMax", val)
	}

	if val, ok := req.Params.MinSqft.Float64(); ok {
		builder.addClause("SearchFilterLivingAreaMin", val)
	}
	if val, ok := req.Params.MaxSqft.Float64(); ok {
		builder.addClause("SearchFilterLivingAreaMax", val)
	}

	if val, ok := req.Params.MinLotSizeAcres.Float64(); ok {
		builder.addClause("SearchFilterLotSizeMin", val)
	}
	if val, ok := req.Params.MaxLotSizeAcres.Float64(); ok {
		builder.addClause("SearchFilterLotSizeMax", val)
	}

	if val, ok := req.Params.MinYearBuilt.Int(); ok {
		builder.addClause("SearchFilterYearBuiltMin", val)
	}
	if val, ok := req.Params.MaxYearBuilt.Int(); ok {
		builder.addClause("SearchFilterYearBuiltMax", val)
	}

	if val, ok := req.Params.MinDOM.Int(); ok {
		builder.addClause("SearchFilterDaysOnMarketMin", val)
	}
	if val, ok := req.Params.MaxDOM.Int(); ok {
		builder.addClause("SearchFilterDaysOnMarketMax", val)
	}
	if val, ok := req.Params.PriceReducedWithinDays.Int(); ok {
		builder.addClause("SearchFilterPriceReducedWithinDays", val)
	}

	if val, ok := req.Params.PoolPrivate.Bool(); ok && val {
		builder.addClause("SearchFilterPoolPrivate")
	}
	if val, ok := req.Params.Waterfront.Bool(); ok && val {
		builder.addClause("SearchFilterWaterfront")
	}
	if val, ok := req.Params.Dock.Bool(); ok && val {
		builder.addClause("SearchFilterDock")
	}
	if val, ok := req.Params.NewConstruction.Bool(); ok && val {
		builder.addClause("SearchFilterNewConstruction")
	}
	if val, ok := req.Params.HasPhotos.Bool(); ok && val {
		builder.addClause("SearchFilterHasPhotos")
	}

	// New SEO filters (Phase 11)
	if val, ok := req.Params.ActiveAdultCommunity.Bool(); ok && val {
		builder.addClause("SearchFilterActiveAdultCommunity")
	}
	if val, ok := req.Params.Association.Bool(); ok && val {
		builder.addClause("SearchFilterAssociation")
	}
	if val, ok := req.Params.Fireplace.Bool(); ok && val {
		builder.addClause("SearchFilterFireplace")
	}
	if val, ok := req.Params.Spa.Bool(); ok && val {
		builder.addClause("SearchFilterSpa")
	}
	if val, ok := req.Params.ForLease.Bool(); ok && val {
		builder.addClause("SearchFilterForLease")
	}
	if val, ok := req.Params.Garage.Bool(); ok && val {
		builder.addClause("SearchFilterGarage")
	}
	if val, ok := req.Params.MinMonthlyFees.Float64(); ok {
		builder.addClause("SearchFilterMinMonthlyFees", val)
	}
	if val, ok := req.Params.MaxMonthlyFees.Float64(); ok {
		builder.addClause("SearchFilterMaxMonthlyFees", val)
	}
	if val, ok := req.Params.MinStories.Int(); ok {
		builder.addClause("SearchFilterStoriesMin", val)
	}
	if val, ok := req.Params.MaxStories.Int(); ok {
		builder.addClause("SearchFilterStoriesMax", val)
	}

	locationClauses := make([]string, 0, 5)
	addLocation := func(queryName string, values []int64) {
		if len(values) == 0 {
			return
		}
		fragment := strings.TrimSpace(builder.registry.SQL(db.QueryName(queryName)))
		fragment = replaceArgs(fragment, builder.argIndex, 1)
		builder.argIndex++
		builder.args = append(builder.args, values)
		fragment = extractClause(fragment)
		if fragment != "" {
			locationClauses = append(locationClauses, fragment)
		}
	}

	addLocation("SearchFilterStateRefIDs", req.LocationFilters.StateRefIDs.Values)
	addLocation("SearchFilterCountyRefIDs", req.LocationFilters.CountyRefIDs.Values)
	addLocation("SearchFilterCityRefIDs", req.LocationFilters.CityRefIDs.Values)
	addLocation("SearchFilterSubdivisionRefIDs", req.LocationFilters.SubdivisionRefIDs.Values)
	addLocation("SearchFilterPostalCodeRefIDs", req.LocationFilters.PostalCodeRefIDs.Values)
	addLocation("SearchFilterElementarySchoolRefIDs", req.LocationFilters.ElementarySchoolRefIDs.Values)
	addLocation("SearchFilterMiddleSchoolRefIDs", req.LocationFilters.MiddleSchoolRefIDs.Values)
	addLocation("SearchFilterHighSchoolRefIDs", req.LocationFilters.HighSchoolRefIDs.Values)
	builder.addGroupOr(locationClauses)

	if req.Params.Geo != nil {
		if lat, lng, radiusMiles, ok := req.Params.Geo.DistanceValues(); ok {
			radiusMeters := radiusMiles * 1609.344
			builder.addClause("SearchFilterDistance", lng, lat, radiusMeters)
		}
		if west, south, east, north, ok := req.Params.Geo.BBoxValues(); ok {
			builder.addClause("SearchFilterBBox", west, south, east, north)
		}
		if wkt, ok := req.Params.Geo.PolygonWKT(); ok {
			builder.addClause("SearchFilterPolygon", wkt)
		}
	}
}

func detectPartitionGroup(statuses []string) string {
	hasActive := false
	hasSold := false
	hasOther := false

	for _, s := range statuses {
		switch s {
		case "Active", "Pending", "Active Under Contract", "Coming Soon", "Incomplete":
			hasActive = true
		case "Closed":
			hasSold = true
		case "Expired", "Withdrawn", "Canceled", "Delete", "Hold":
			hasOther = true
		default:
			// If unknown status, assume unsafe to prune
			return ""
		}
	}

	if hasActive && !hasSold && !hasOther {
		return "active"
	}
	if hasSold && !hasActive && !hasOther {
		return "closed"
	}
	if hasOther && !hasActive && !hasSold {
		return "other"
	}

	return ""
}

type sortSpec struct {
	key              string
	defaultDirection string
	selectAsc        string
	selectDesc       string
	cursorAsc        string
	cursorAscNull    string
	cursorDesc       string
	cursorDescNull   string
	isDistance       bool
}

func (s sortSpec) selectQuery(dir string) string {
	if s.isDistance {
		return s.selectAsc
	}
	if dir == "asc" {
		return s.selectAsc
	}
	return s.selectDesc
}

func resolveSort(req SearchRequest, distanceAvailable bool) (sortSpec, string) {
	defaultSpec := sortSpecByKey()["on_market_date"]
	sortKey := req.Params.NormalizeSort("on_market_date")
	spec, ok := sortSpecByKey()[sortKey]
	if !ok {
		spec = defaultSpec
	}

	if sortKey == "distance" {
		if !distanceAvailable {
			spec = defaultSpec
		}
	}

	direction := req.Params.NormalizeSortDir(spec.defaultDirection)
	if spec.isDistance {
		direction = "asc"
	}
	return spec, direction
}

func sortSpecByKey() map[string]sortSpec {
	return map[string]sortSpec{
		"list_price": {
			key:              "list_price",
			defaultDirection: "desc",
			selectAsc:        "SearchSelectListPriceAsc",
			selectDesc:       "SearchSelectListPriceDesc",
			cursorAsc:        "SearchCursorListPriceAsc",
			cursorAscNull:    "SearchCursorListPriceAscNull",
			cursorDesc:       "SearchCursorListPriceDesc",
			cursorDescNull:   "SearchCursorListPriceDescNull",
		},
		"on_market_date": {
			key:              "on_market_date",
			defaultDirection: "desc",
			selectAsc:        "SearchSelectOnMarketDateAsc",
			selectDesc:       "SearchSelectOnMarketDateDesc",
			cursorAsc:        "SearchCursorOnMarketDateAsc",
			cursorAscNull:    "SearchCursorOnMarketDateAscNull",
			cursorDesc:       "SearchCursorOnMarketDateDesc",
			cursorDescNull:   "SearchCursorOnMarketDateDescNull",
		},
		"year_built": {
			key:              "year_built",
			defaultDirection: "desc",
			selectAsc:        "SearchSelectYearBuiltAsc",
			selectDesc:       "SearchSelectYearBuiltDesc",
			cursorAsc:        "SearchCursorYearBuiltAsc",
			cursorAscNull:    "SearchCursorYearBuiltAscNull",
			cursorDesc:       "SearchCursorYearBuiltDesc",
			cursorDescNull:   "SearchCursorYearBuiltDescNull",
		},
		"living_area": {
			key:              "living_area",
			defaultDirection: "desc",
			selectAsc:        "SearchSelectLivingAreaAsc",
			selectDesc:       "SearchSelectLivingAreaDesc",
			cursorAsc:        "SearchCursorLivingAreaAsc",
			cursorAscNull:    "SearchCursorLivingAreaAscNull",
			cursorDesc:       "SearchCursorLivingAreaDesc",
			cursorDescNull:   "SearchCursorLivingAreaDescNull",
		},
		"lot_size_acres": {
			key:              "lot_size_acres",
			defaultDirection: "desc",
			selectAsc:        "SearchSelectLotSizeAsc",
			selectDesc:       "SearchSelectLotSizeDesc",
			cursorAsc:        "SearchCursorLotSizeAsc",
			cursorAscNull:    "SearchCursorLotSizeAscNull",
			cursorDesc:       "SearchCursorLotSizeDesc",
			cursorDescNull:   "SearchCursorLotSizeDescNull",
		},
		"bedrooms_total": {
			key:              "bedrooms_total",
			defaultDirection: "desc",
			selectAsc:        "SearchSelectBedroomsAsc",
			selectDesc:       "SearchSelectBedroomsDesc",
			cursorAsc:        "SearchCursorBedroomsAsc",
			cursorAscNull:    "SearchCursorBedroomsAscNull",
			cursorDesc:       "SearchCursorBedroomsDesc",
			cursorDescNull:   "SearchCursorBedroomsDescNull",
		},
		"bathrooms_total": {
			key:              "bathrooms_total",
			defaultDirection: "desc",
			selectAsc:        "SearchSelectBathroomsAsc",
			selectDesc:       "SearchSelectBathroomsDesc",
			cursorAsc:        "SearchCursorBathroomsAsc",
			cursorAscNull:    "SearchCursorBathroomsAscNull",
			cursorDesc:       "SearchCursorBathroomsDesc",
			cursorDescNull:   "SearchCursorBathroomsDescNull",
		},
		"distance": {
			key:              "distance",
			defaultDirection: "asc",
			selectAsc:        "SearchSelectDistanceAsc",
			selectDesc:       "SearchSelectDistanceAsc",
			cursorAsc:        "SearchCursorDistanceAsc",
			cursorAscNull:    "SearchCursorDistanceAscNull",
			cursorDesc:       "SearchCursorDistanceAsc",
			cursorDescNull:   "SearchCursorDistanceAscNull",
			isDistance:       true,
		},
	}
}

func buildCursorClause(req SearchRequest, spec sortSpec, dir string, argIndex int, registry *db.Registry) (string, []any, error) {
	if req.Page.Cursor == "" {
		return "", nil, nil
	}
	rawCursor, err := DecodeCursor(req.Page.Cursor)
	if err != nil {
		return "", nil, cursorValidationError(err.Error())
	}

	cursorSortKey, isNull, err := parseCursorSortKey(rawCursor.SortKey, spec.key)
	if err != nil {
		return "", nil, cursorValidationError(err.Error())
	}

	var queryName string
	if dir == "asc" {
		if isNull {
			queryName = spec.cursorAscNull
		} else {
			queryName = spec.cursorAsc
		}
	} else {
		if isNull {
			queryName = spec.cursorDescNull
		} else {
			queryName = spec.cursorDesc
		}
	}

	cursorClause := strings.TrimSpace(registry.SQL(db.QueryName(queryName)))
	if isNull {
		cursorClause = replaceArgs(cursorClause, argIndex, 1)
		cursorClause = extractClause(cursorClause)
		return cursorClause, []any{rawCursor.PK}, nil
	}
	if spec.isDistance {
		cursorClause = replaceDistanceCursorArgs(cursorClause, argIndex)
	} else {
		cursorClause = replaceArgs(cursorClause, argIndex, 2)
	}
	cursorClause = extractClause(cursorClause)
	return cursorClause, []any{cursorSortKey, rawCursor.PK}, nil
}

func parseCursorSortKey(raw json.RawMessage, key string) (any, bool, error) {
	trimmed := strings.TrimSpace(string(raw))
	if len(raw) == 0 || trimmed == "" || trimmed == "null" {
		return nil, true, nil
	}

	switch key {
	case "list_price", "living_area", "lot_size_acres", "distance":
		var val float64
		if err := json.Unmarshal(raw, &val); err != nil {
			return nil, false, fmt.Errorf("invalid cursor sort_key")
		}
		return val, false, nil
	case "year_built", "days_on_market", "bedrooms_total", "bathrooms_total":
		var val float64
		if err := json.Unmarshal(raw, &val); err != nil {
			return nil, false, fmt.Errorf("invalid cursor sort_key")
		}
		if val != float64(int64(val)) {
			return nil, false, fmt.Errorf("invalid cursor sort_key")
		}
		return int64(val), false, nil
	case "on_market_date":
		var val string
		if err := json.Unmarshal(raw, &val); err != nil {
			return nil, false, fmt.Errorf("invalid cursor sort_key")
		}
		parsed, err := time.Parse(time.RFC3339Nano, val)
		if err != nil {
			return nil, false, fmt.Errorf("invalid cursor sort_key")
		}
		return parsed, false, nil
	default:
		return nil, false, fmt.Errorf("unsupported cursor sort_key")
	}
}

func cursorValidationError(message string) error {
	err := newValidationError()
	err.Fields["page.cursor"] = message
	return err
}

func extractSortKey(row searchRow, spec sortSpec) (any, error) {
	switch spec.key {
	case "list_price":
		if !row.ListPrice.Valid {
			return nil, nil
		}
		return numericToFloat64(row.ListPrice), nil
	case "on_market_date":
		if !row.OnMarketDate.Valid {
			return nil, nil
		}
		return row.OnMarketDate.Time.Format(time.RFC3339Nano), nil
	case "year_built":
		if !row.YearBuilt.Valid {
			return nil, nil
		}
		return int64(row.YearBuilt.Int32), nil
	case "living_area":
		if !row.LivingArea.Valid {
			return nil, nil
		}
		return float64(row.LivingArea.Int32), nil
	case "lot_size_acres":
		if !row.LotSizeAcres.Valid {
			return nil, nil
		}
		return numericToFloat64(row.LotSizeAcres), nil
	case "bedrooms_total":
		if !row.BedroomsTotal.Valid {
			return nil, nil
		}
		return int64(row.BedroomsTotal.Int32), nil
	case "bathrooms_total":
		if !row.BathroomsTotal.Valid {
			return nil, nil
		}
		return int64(row.BathroomsTotal.Int32), nil
	case "distance":
		if !row.DistanceMeters.Valid {
			return nil, nil
		}
		return row.DistanceMeters.Float64, nil
	default:
		return nil, fmt.Errorf("unsupported sort key")
	}
}
