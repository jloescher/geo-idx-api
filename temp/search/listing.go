package search

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/sync/errgroup"

	"github.com/xotec-solutions/xotec-datalayer/src/internal/db"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/mls"
)

// ListingBasic contains minimal listing data for historical display.
type ListingBasic struct {
	ListingID       string       `json:"listing_id"`
	StandardStatus  *string      `json:"standard_status,omitempty"`
	ListPrice       *float64     `json:"list_price,omitempty"`
	StreetNumber    *string      `json:"street_number,omitempty"`
	StreetName      *string      `json:"street_name,omitempty"`
	City            *string      `json:"city,omitempty"`
	StateOrProvince *string      `json:"state_or_province,omitempty"`
	PostalCode      *string      `json:"postal_code,omitempty"`
	BedroomsTotal   *int32       `json:"bedrooms_total,omitempty"`
	BathroomsTotal  *float64     `json:"bathrooms_total,omitempty"`
	LivingArea      *float64     `json:"living_area,omitempty"`
	PrimaryImage    PrimaryImage `json:"primary_image"`
	OnMarketDate    *time.Time   `json:"on_market_date,omitempty"`
	CloseDate       *time.Time   `json:"close_date,omitempty"`
}

// listingBasicRow maps the database row from GetListingBasics query.
type listingBasicRow struct {
	ListingID             pgtype.Text        `db:"listing_id"`
	StandardStatus        pgtype.Text        `db:"standard_status"`
	ListPrice             pgtype.Numeric     `db:"list_price"`
	StreetNumber          pgtype.Text        `db:"street_number"`
	StreetName            pgtype.Text        `db:"street_name"`
	City                  pgtype.Text        `db:"city"`
	StateOrProvince       pgtype.Text        `db:"state_or_province"`
	PostalCode            pgtype.Text        `db:"postal_code"`
	BedroomsTotal         pgtype.Int4        `db:"bedrooms_total"`
	BathroomsTotal        pgtype.Int4        `db:"bathrooms_total"`
	LivingArea            pgtype.Int4        `db:"living_area"`
	OnMarketDate          pgtype.Timestamptz `db:"on_market_date"`
	CloseDate             pgtype.Timestamptz `db:"close_date"`
	PartitionGroup        pgtype.Text        `db:"partition_group"`
	PrimaryMediaKey       pgtype.Text        `db:"primary_media_key"`
	PrimaryHostedKey      pgtype.Text        `db:"primary_hosted_key"`
	PrimaryMediaOptimized pgtype.Bool        `db:"primary_media_optimized"`
}

// ListingDetails includes all listing data across tables.
type ListingDetails struct {
	ListingID     string          `json:"listing_id"`
	Core          json.RawMessage `json:"core"`
	Extended      json.RawMessage `json:"extended"`
	Extra         json.RawMessage `json:"extra"`
	Media         json.RawMessage `json:"media"`
	Rooms         json.RawMessage `json:"rooms"`
	UnitTypes     json.RawMessage `json:"unit_types"`
	PriceHistory  json.RawMessage `json:"price_history"`
	StatusHistory json.RawMessage `json:"status_history"`
}

// GetListingDetails fetches the listing details by listing ID.
// It attempts to resolve the partition group first for optimized lookup.
func (s *Store) GetListingDetails(ctx context.Context, listingID string) (ListingDetails, bool, error) {
	// 1. Try to resolve partition group from map
	var partitionGroup string
	// Check if query exists first (in case migration didn't run, though it should have)
	// We assume it exists as per plan.
	err := s.Pool.QueryRow(ctx, s.Registry.SQL("GetListingPartition"), listingID).Scan(&partitionGroup)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			// Log error but fallback? No, DB error is bad.
			// However, if the query doesn't exist in registry (partial deploy), it panics.
			// We assume registry is consistent.
			return ListingDetails{}, false, fmt.Errorf("lookup partition: %w", err)
		}
		// Not found in map -> Try legacy lookup (maybe it's not in map yet or strictly legacy)
		return s.getListingDetailsLegacy(ctx, listingID)
	}

	// Found in map -> Use optimized partitioned lookup
	return s.getListingDetailsPartitioned(ctx, listingID, partitionGroup)
}

func (s *Store) getListingDetailsPartitioned(ctx context.Context, listingID, partitionGroup string) (ListingDetails, bool, error) {
	core, err := s.fetchJSONPartitioned(ctx, "GetListingCoreJSONPartitioned", listingID, partitionGroup)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ListingDetails{}, false, nil
		}
		return ListingDetails{}, false, fmt.Errorf("fetch core listing (partitioned): %w", err)
	}

	g, gCtx := errgroup.WithContext(ctx)

	var extended, extra, media, rooms, unitTypes, priceHistory, statusHistory json.RawMessage

	g.Go(func() error {
		var err error
		extended, err = s.fetchJSONOptionalPartitioned(gCtx, "GetListingExtendedJSONPartitioned", listingID, partitionGroup)
		if err != nil {
			return fmt.Errorf("fetch extended listing: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		var err error
		extra, err = s.fetchJSONOptionalPartitioned(gCtx, "GetListingExtraJSONPartitioned", listingID, partitionGroup)
		if err != nil {
			return fmt.Errorf("fetch extra listing: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		var err error
		media, err = s.fetchJSONPartitioned(gCtx, "GetListingMediaJSONPartitioned", listingID, partitionGroup)
		if err != nil {
			return fmt.Errorf("fetch media: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		var err error
		rooms, err = s.fetchJSONPartitioned(gCtx, "GetListingRoomsJSONPartitioned", listingID, partitionGroup)
		if err != nil {
			return fmt.Errorf("fetch rooms: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		var err error
		unitTypes, err = s.fetchJSONPartitioned(gCtx, "GetListingUnitTypesJSONPartitioned", listingID, partitionGroup)
		if err != nil {
			return fmt.Errorf("fetch unit types: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		var err error
		priceHistory, err = s.fetchJSONPartitionedListing(gCtx, "GetListingPriceHistoryJSONPartitioned", listingID)
		if err != nil {
			return fmt.Errorf("fetch price history: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		var err error
		statusHistory, err = s.fetchJSONPartitionedListing(gCtx, "GetListingStatusHistoryJSONPartitioned", listingID)
		if err != nil {
			return fmt.Errorf("fetch status history: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return ListingDetails{}, false, err
	}

	media, err = s.decorateListingMedia(listingID, media)
	if err != nil {
		return ListingDetails{}, false, fmt.Errorf("decorate media: %w", err)
	}

	core = injectDaysOnMarket(core)
	core = stripJSONListingPrefix(core)
	extended = stripJSONListingPrefix(extended)
	extra = stripJSONListingPrefix(extra)
	rooms = stripJSONArrayListingPrefix(rooms)
	unitTypes = stripJSONArrayListingPrefix(unitTypes)
	priceHistory = stripJSONArrayListingPrefix(priceHistory)
	statusHistory = stripJSONArrayListingPrefix(statusHistory)

	return ListingDetails{
		ListingID:     mls.StripPrefix(listingID),
		Core:          core,
		Extended:      extended,
		Extra:         extra,
		Media:         media,
		Rooms:         rooms,
		UnitTypes:     unitTypes,
		PriceHistory:  priceHistory,
		StatusHistory: statusHistory,
	}, true, nil
}

func (s *Store) getListingDetailsLegacy(ctx context.Context, listingID string) (ListingDetails, bool, error) {
	core, err := s.fetchJSON(ctx, "GetListingCoreJSON", listingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ListingDetails{}, false, nil
		}
		return ListingDetails{}, false, fmt.Errorf("fetch core listing: %w", err)
	}

	g, gCtx := errgroup.WithContext(ctx)

	var extended, extra, media, rooms, unitTypes, priceHistory, statusHistory json.RawMessage

	g.Go(func() error {
		var err error
		extended, err = s.fetchJSONOptional(gCtx, "GetListingExtendedJSON", listingID)
		if err != nil {
			return fmt.Errorf("fetch extended listing: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		var err error
		extra, err = s.fetchJSONOptional(gCtx, "GetListingExtraJSON", listingID)
		if err != nil {
			return fmt.Errorf("fetch extra listing: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		var err error
		media, err = s.fetchJSON(gCtx, "GetListingMediaJSON", listingID)
		if err != nil {
			return fmt.Errorf("fetch media: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		var err error
		rooms, err = s.fetchJSON(gCtx, "GetListingRoomsJSON", listingID)
		if err != nil {
			return fmt.Errorf("fetch rooms: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		var err error
		unitTypes, err = s.fetchJSON(gCtx, "GetListingUnitTypesJSON", listingID)
		if err != nil {
			return fmt.Errorf("fetch unit types: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		var err error
		priceHistory, err = s.fetchJSON(gCtx, "GetListingPriceHistoryJSON", listingID)
		if err != nil {
			return fmt.Errorf("fetch price history: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		var err error
		statusHistory, err = s.fetchJSON(gCtx, "GetListingStatusHistoryJSON", listingID)
		if err != nil {
			return fmt.Errorf("fetch status history: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return ListingDetails{}, false, err
	}

	media, err = s.decorateListingMedia(listingID, media)
	if err != nil {
		return ListingDetails{}, false, fmt.Errorf("decorate media: %w", err)
	}

	core = injectDaysOnMarket(core)
	core = stripJSONListingPrefix(core)
	extended = stripJSONListingPrefix(extended)
	extra = stripJSONListingPrefix(extra)
	rooms = stripJSONArrayListingPrefix(rooms)
	unitTypes = stripJSONArrayListingPrefix(unitTypes)
	priceHistory = stripJSONArrayListingPrefix(priceHistory)
	statusHistory = stripJSONArrayListingPrefix(statusHistory)

	return ListingDetails{
		ListingID:     mls.StripPrefix(listingID),
		Core:          core,
		Extended:      extended,
		Extra:         extra,
		Media:         media,
		Rooms:         rooms,
		UnitTypes:     unitTypes,
		PriceHistory:  priceHistory,
		StatusHistory: statusHistory,
	}, true, nil
}

const maxListingBasicsBatch = 50

// GetListingBasics fetches minimal listing data for a batch of listing IDs.
// Returns listings found (up to maxListingBasicsBatch per call).
func (s *Store) GetListingBasics(ctx context.Context, listingIDs []string) ([]ListingBasic, error) {
	if len(listingIDs) == 0 {
		return []ListingBasic{}, nil
	}

	// Cap the batch size for safety
	if len(listingIDs) > maxListingBasicsBatch {
		listingIDs = listingIDs[:maxListingBasicsBatch]
	}

	query := s.Registry.SQL("GetListingBasics")
	rows, err := s.Pool.Query(ctx, query, listingIDs)
	if err != nil {
		return nil, fmt.Errorf("query listing basics: %w", err)
	}
	defer rows.Close()

	var results []ListingBasic
	for rows.Next() {
		var row listingBasicRow
		if err := rows.Scan(
			&row.ListingID,
			&row.StandardStatus,
			&row.ListPrice,
			&row.StreetNumber,
			&row.StreetName,
			&row.City,
			&row.StateOrProvince,
			&row.PostalCode,
			&row.BedroomsTotal,
			&row.BathroomsTotal,
			&row.LivingArea,
			&row.OnMarketDate,
			&row.CloseDate,
			&row.PartitionGroup,
			&row.PrimaryMediaKey,
			&row.PrimaryHostedKey,
			&row.PrimaryMediaOptimized,
		); err != nil {
			return nil, fmt.Errorf("scan listing basic row: %w", err)
		}

		basic := mapListingBasicRow(row, s.MediaCDNHost)
		results = append(results, basic)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate listing basics: %w", err)
	}

	return results, nil
}

func mapListingBasicRow(row listingBasicRow, mediaCDNHost string) ListingBasic {
	rawListingID := textValue(row.ListingID)
	mediaKey := resolvePrimaryMediaKey(rawListingID, row.PrimaryMediaKey, row.PrimaryHostedKey)

	// Build primary image with sources matching search result format
	listingID := mls.StripPrefix(rawListingID)
	primaryImage := PrimaryImage{}
	if mediaKey != "" {
		optimized := row.PrimaryMediaOptimized.Valid && row.PrimaryMediaOptimized.Bool
		primaryImage = PrimaryImage{
			MediaKey:    &mediaKey,
			Sources:     buildPrimaryImageSources(mediaCDNHost, listingID, mediaKey),
			IsOptimized: &optimized,
		}
	}

	// Convert bathrooms from int to float for API consistency
	var bathroomsTotal *float64
	if row.BathroomsTotal.Valid {
		val := float64(row.BathroomsTotal.Int32)
		bathroomsTotal = &val
	}

	// Convert living area from int to float for API consistency
	var livingArea *float64
	if row.LivingArea.Valid {
		val := float64(row.LivingArea.Int32)
		livingArea = &val
	}

	return ListingBasic{
		ListingID:       listingID,
		StandardStatus:  textPtr(row.StandardStatus),
		ListPrice:       numericPtr(row.ListPrice),
		StreetNumber:    textPtr(row.StreetNumber),
		StreetName:      textPtr(row.StreetName),
		City:            textPtr(row.City),
		StateOrProvince: textPtr(row.StateOrProvince),
		PostalCode:      textPtr(row.PostalCode),
		BedroomsTotal:   int32Ptr(row.BedroomsTotal),
		BathroomsTotal:  bathroomsTotal,
		LivingArea:      livingArea,
		PrimaryImage:    primaryImage,
		OnMarketDate:    timePtr(row.OnMarketDate),
		CloseDate:       timePtr(row.CloseDate),
	}
}

func int32Ptr(n pgtype.Int4) *int32 {
	if !n.Valid {
		return nil
	}
	val := n.Int32
	return &val
}

func (s *Store) fetchJSON(ctx context.Context, queryName string, listingID string) (json.RawMessage, error) {
	query := s.Registry.SQL(db.QueryName(queryName))
	var raw json.RawMessage
	if err := s.Pool.QueryRow(ctx, query, listingID).Scan(&raw); err != nil {
		return nil, err
	}
	if raw == nil {
		return json.RawMessage("null"), nil
	}
	return raw, nil
}

func (s *Store) fetchJSONOptional(ctx context.Context, queryName string, listingID string) (json.RawMessage, error) {
	raw, err := s.fetchJSON(ctx, queryName, listingID)
	if errors.Is(err, pgx.ErrNoRows) {
		return json.RawMessage("null"), nil
	}
	return raw, err
}

func (s *Store) fetchJSONPartitioned(ctx context.Context, queryName string, listingID, partitionGroup string) (json.RawMessage, error) {
	query := s.Registry.SQL(db.QueryName(queryName))
	var raw json.RawMessage
	if err := s.Pool.QueryRow(ctx, query, listingID, partitionGroup).Scan(&raw); err != nil {
		return nil, err
	}
	if raw == nil {
		return json.RawMessage("null"), nil
	}
	return raw, nil
}

func (s *Store) fetchJSONPartitionedListing(ctx context.Context, queryName string, listingID string) (json.RawMessage, error) {
	query := s.Registry.SQL(db.QueryName(queryName))
	var raw json.RawMessage
	if err := s.Pool.QueryRow(ctx, query, listingID).Scan(&raw); err != nil {
		return nil, err
	}
	if raw == nil {
		return json.RawMessage("null"), nil
	}
	return raw, nil
}

func (s *Store) fetchJSONOptionalPartitioned(ctx context.Context, queryName string, listingID, partitionGroup string) (json.RawMessage, error) {
	raw, err := s.fetchJSONPartitioned(ctx, queryName, listingID, partitionGroup)
	if errors.Is(err, pgx.ErrNoRows) {
		return json.RawMessage("null"), nil
	}
	return raw, err
}

func (s *Store) decorateListingMedia(listingID string, raw json.RawMessage) (json.RawMessage, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return raw, nil
	}

	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("decode media json: %w", err)
	}

	for _, item := range items {
		delete(item, "media_url")

		mediaKey, _ := item["media_key"].(string)
		hostedKey, _ := item["hosted_key"].(string)
		resolvedKey := resolveMediaKeyForRecord(mediaKey, hostedKey)
		if resolvedKey == "" {
			stripMediaItemListingPrefix(item)
			delete(item, "sources")
			continue
		}

		effectiveListingID := listingID
		if strings.TrimSpace(effectiveListingID) == "" {
			if candidate, ok := item["listing_id"].(string); ok {
				effectiveListingID = candidate
			}
		}

		sources := buildPrimaryImageSources(s.MediaCDNHost, effectiveListingID, resolvedKey)
		if sources == nil || sources.AVIF == nil || sources.WebP == nil {
			stripMediaItemListingPrefix(item)
			delete(item, "sources")
			continue
		}
		item["sources"] = map[string]any{
			"avif": *sources.AVIF,
			"webp": *sources.WebP,
		}
		stripMediaItemListingPrefix(item)
	}

	encoded, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("encode media json: %w", err)
	}
	return encoded, nil
}

func injectDaysOnMarket(core json.RawMessage) json.RawMessage {
	trimmed := strings.TrimSpace(string(core))
	if trimmed == "" || trimmed == "null" {
		return core
	}

	var m map[string]any
	if err := json.Unmarshal(core, &m); err != nil {
		return core
	}

	raw, ok := m["on_market_date"]
	if !ok || raw == nil {
		return core
	}
	dateStr, ok := raw.(string)
	if !ok || dateStr == "" {
		return core
	}
	omd, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return core
	}

	now := time.Now().In(nyLoc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, nyLoc)
	omdNY := omd.In(nyLoc)
	omdDay := time.Date(omdNY.Year(), omdNY.Month(), omdNY.Day(), 0, 0, 0, 0, nyLoc)
	dom := int(today.Sub(omdDay).Hours() / 24)
	if dom < 0 {
		dom = 0
	}
	m["days_on_market"] = dom

	out, err := json.Marshal(m)
	if err != nil {
		return core
	}
	return out
}

// stripJSONListingPrefix strips the MLS prefix from the "listing_id" key
// inside a single JSON object blob. No-op on null, empty, or parse error.
func stripJSONListingPrefix(raw json.RawMessage) json.RawMessage {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return raw
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw
	}
	lid, ok := m["listing_id"].(string)
	if !ok || lid == "" {
		return raw
	}
	m["listing_id"] = mls.StripPrefix(lid)
	out, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return out
}

// stripJSONArrayListingPrefix strips the MLS prefix from "listing_id" keys
// inside each object in a JSON array blob. No-op on null, empty, or parse error.
func stripJSONArrayListingPrefix(raw json.RawMessage) json.RawMessage {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return raw
	}
	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err != nil {
		return raw
	}
	changed := false
	for _, item := range items {
		if lid, ok := item["listing_id"].(string); ok {
			item["listing_id"] = mls.StripPrefix(lid)
			changed = true
		}
	}
	if !changed {
		return raw
	}
	out, err := json.Marshal(items)
	if err != nil {
		return raw
	}
	return out
}

func stripMediaItemListingPrefix(item map[string]any) {
	if lid, ok := item["listing_id"].(string); ok {
		item["listing_id"] = mls.StripPrefix(lid)
	}
}

func resolveMediaKeyForRecord(mediaKey string, hostedKey string) string {
	value := strings.TrimSpace(mediaKey)
	if value != "" {
		return value
	}
	value = strings.TrimSpace(hostedKey)
	if value == "" {
		return ""
	}
	value = stripMediaExtension(value)
	if strings.Contains(value, "/") {
		parts := strings.Split(value, "/")
		value = parts[len(parts)-1]
	}
	return strings.TrimSpace(value)
}
