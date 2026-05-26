package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/mls"
)

// HydrateStats summarizes mirror persist for a replica chunk.
type HydrateStats struct {
	RowsReceived int
	Upserted     int
	Deleted      int
	Skipped      int
}

type coordPair struct {
	datasetSlug string
	listingKey  string
	lng, lat    float64
}

// ListingMirrorWriter upserts rows into listings mirror with PostGIS coordinates.
type ListingMirrorWriter struct {
	db           *repository.DB
	resolver     *mls.ResoFieldResolver
	upsertChunk  int
	sparkExpand  string
	bridgeExpand string
}

func NewListingMirrorWriter(db *repository.DB, upsertChunk int, sparkExpand, bridgeExpand string) *ListingMirrorWriter {
	if upsertChunk <= 0 {
		upsertChunk = 250
	}
	return &ListingMirrorWriter{
		db:           db,
		resolver:     mls.NewResoFieldResolver(),
		upsertChunk:  upsertChunk,
		sparkExpand:  sparkExpand,
		bridgeExpand: bridgeExpand,
	}
}

func (w *ListingMirrorWriter) expandKeys(provider mls.MirrorProvider) []string {
	return mls.PersistExpandKeys(provider, w.sparkExpand, w.bridgeExpand)
}

// HydrateReplicaBatch maps RESO rows to indexed listings columns (Active/Pending upsert; others delete).
func (w *ListingMirrorWriter) HydrateReplicaBatch(
	ctx context.Context,
	replicationDataset string,
	provider mls.MirrorProvider,
	rows []json.RawMessage,
) (HydrateStats, error) {
	stats := HydrateStats{RowsReceived: len(rows)}

	var pending []mls.ListingRecord
	var coords []coordPair
	nullCoordKeys := make(map[string]map[string]bool)
	deletes := make(map[string]map[string]bool)

	flush := func() error {
		if len(pending) == 0 && len(coords) == 0 && len(nullCoordKeys) == 0 {
			return nil
		}
		tx, err := w.db.Pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		for _, rec := range pending {
			if err := upsertListing(ctx, tx, rec); err != nil {
				return err
			}
		}
		if err := flushCoordinates(ctx, tx, coords); err != nil {
			return err
		}
		if err := flushNullCoordinates(ctx, tx, nullCoordKeys); err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		pending = nil
		coords = nil
		nullCoordKeys = make(map[string]map[string]bool)
		return nil
	}

	for _, raw := range rows {
		var row map[string]any
		if err := json.Unmarshal(raw, &row); err != nil {
			stats.Skipped++
			continue
		}

		rec, action := mls.BuildListingRecord(replicationDataset, provider, row, raw, w.resolver, w.expandKeys(provider))
		switch action {
		case mls.RowActionSkip:
			stats.Skipped++
			continue
		case mls.RowActionDelete:
			if deletes[rec.DatasetSlug] == nil {
				deletes[rec.DatasetSlug] = make(map[string]bool)
			}
			deletes[rec.DatasetSlug][rec.ListingKey] = true
			stats.Deleted++
			continue
		}

		pending = append(pending, rec)
		stats.Upserted++

		if rec.Latitude != nil && rec.Longitude != nil {
			coords = append(coords, coordPair{
				datasetSlug: rec.DatasetSlug,
				listingKey:  rec.ListingKey,
				lng:         *rec.Longitude,
				lat:         *rec.Latitude,
			})
		} else {
			if nullCoordKeys[rec.DatasetSlug] == nil {
				nullCoordKeys[rec.DatasetSlug] = make(map[string]bool)
			}
			nullCoordKeys[rec.DatasetSlug][rec.ListingKey] = true
		}

		if len(pending) >= w.upsertChunk {
			if err := flush(); err != nil {
				return stats, err
			}
		}
	}

	if err := flush(); err != nil {
		return stats, err
	}
	if err := w.flushDeletes(ctx, deletes); err != nil {
		return stats, err
	}
	return stats, nil
}

func nullableJSONB(payload json.RawMessage, present bool) any {
	if !present {
		return nil
	}
	return payload
}

func upsertListing(ctx context.Context, tx pgx.Tx, rec mls.ListingRecord) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO listings (
			dataset_slug, listing_key, mls_listing_id, standard_status,
			list_price, bedrooms_total, bathrooms_total_decimal, living_area, lot_size_acres,
			year_built, stories_total, city, county_or_parish, postal_code, state_or_province,
			property_type, property_sub_type, on_market_date, close_date,
			modification_timestamp, price_change_timestamp,
			previous_list_price, flood_zone_code, estimated_total_monthly_fees, low_risk_flood_zone_yn,
			latitude, longitude, coordinates,
			waterfront_yn, pool_private_yn, dock_yn, new_construction_yn, garage_yn,
			association_yn, spa_yn, fireplace_yn, senior_community_yn,
			subdivision_name, elementary_school, middle_or_junior_school, high_school,
			special_listing_conditions, raw_data, media, unit, room, open_house, custom_fields,
			street_number, street_name, list_agent_mls_id, list_office_mls_id,
			mirror_persisted_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19,
			$20, $21,
			$22, $23, $24, $25,
			$26, $27, NULL,
			$28, $29, $30, $31, $32,
			$33, $34, $35, $36,
			$37, $38, $39, $40,
			$41, $42, $43, $44, $45, $46, $47,
			$48, $49, $50, $51,
			NOW(), NOW(), NOW()
		)
		ON CONFLICT (dataset_slug, listing_key) DO UPDATE SET
			mls_listing_id = EXCLUDED.mls_listing_id,
			standard_status = EXCLUDED.standard_status,
			list_price = EXCLUDED.list_price,
			bedrooms_total = EXCLUDED.bedrooms_total,
			bathrooms_total_decimal = EXCLUDED.bathrooms_total_decimal,
			living_area = EXCLUDED.living_area,
			lot_size_acres = EXCLUDED.lot_size_acres,
			year_built = EXCLUDED.year_built,
			stories_total = EXCLUDED.stories_total,
			city = EXCLUDED.city,
			county_or_parish = EXCLUDED.county_or_parish,
			postal_code = EXCLUDED.postal_code,
			state_or_province = EXCLUDED.state_or_province,
			property_type = EXCLUDED.property_type,
			property_sub_type = EXCLUDED.property_sub_type,
			on_market_date = EXCLUDED.on_market_date,
			close_date = EXCLUDED.close_date,
			modification_timestamp = EXCLUDED.modification_timestamp,
			price_change_timestamp = EXCLUDED.price_change_timestamp,
			previous_list_price = EXCLUDED.previous_list_price,
			flood_zone_code = EXCLUDED.flood_zone_code,
			estimated_total_monthly_fees = EXCLUDED.estimated_total_monthly_fees,
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			waterfront_yn = EXCLUDED.waterfront_yn,
			pool_private_yn = EXCLUDED.pool_private_yn,
			dock_yn = EXCLUDED.dock_yn,
			new_construction_yn = EXCLUDED.new_construction_yn,
			garage_yn = EXCLUDED.garage_yn,
			association_yn = EXCLUDED.association_yn,
			spa_yn = EXCLUDED.spa_yn,
			fireplace_yn = EXCLUDED.fireplace_yn,
			senior_community_yn = EXCLUDED.senior_community_yn,
			subdivision_name = EXCLUDED.subdivision_name,
			elementary_school = EXCLUDED.elementary_school,
			middle_or_junior_school = EXCLUDED.middle_or_junior_school,
			high_school = EXCLUDED.high_school,
			special_listing_conditions = EXCLUDED.special_listing_conditions,
			raw_data = EXCLUDED.raw_data,
			media = COALESCE(EXCLUDED.media, listings.media),
			unit = COALESCE(EXCLUDED.unit, listings.unit),
			room = COALESCE(EXCLUDED.room, listings.room),
			open_house = COALESCE(EXCLUDED.open_house, listings.open_house),
			custom_fields = EXCLUDED.custom_fields,
			street_number = EXCLUDED.street_number,
			street_name = EXCLUDED.street_name,
			list_agent_mls_id = EXCLUDED.list_agent_mls_id,
			list_office_mls_id = EXCLUDED.list_office_mls_id,
			mirror_persisted_at = NOW(),
			updated_at = NOW()
	`,
		rec.DatasetSlug, rec.ListingKey, rec.MlsListingID, rec.StandardStatus,
		rec.ListPrice, rec.BedroomsTotal, rec.BathroomsTotalDecimal, rec.LivingArea, rec.LotSizeAcres,
		rec.YearBuilt, rec.StoriesTotal, rec.City, rec.CountyOrParish, rec.PostalCode, rec.StateOrProvince,
		rec.PropertyType, rec.PropertySubType, rec.OnMarketDate, rec.CloseDate,
		rec.ModificationTimestamp, rec.PriceChangeTimestamp,
		rec.PreviousListPrice, rec.FloodZoneCode, rec.EstimatedTotalMonthlyFees, rec.LowRiskFloodZoneYN,
		rec.Latitude, rec.Longitude,
		rec.WaterfrontYN, rec.PoolPrivateYN, rec.DockYN, rec.NewConstructionYN, rec.GarageYN,
		rec.AssociationYN, rec.SpaYN, rec.FireplaceYN, rec.SeniorCommunityYN,
		rec.SubdivisionName, rec.ElementarySchool, rec.MiddleOrJuniorSchool, rec.HighSchool,
		rec.SpecialListingConditions, rec.RawData,
		nullableJSONB(rec.Media, rec.HasMedia),
		nullableJSONB(rec.Unit, rec.HasUnit),
		nullableJSONB(rec.Room, rec.HasRoom),
		nullableJSONB(rec.OpenHouse, rec.HasOpenHouse),
		rec.CustomFields,
		rec.StreetNumber, rec.StreetName, rec.ListAgentMlsID, rec.ListOfficeMlsID,
	)
	if err != nil {
		return wrapListingPersistErr(rec, err)
	}
	return nil
}

func wrapListingPersistErr(rec mls.ListingRecord, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "22003" {
		return fmt.Errorf("listing persist numeric overflow dataset=%s listing_key=%s: %w",
			rec.DatasetSlug, rec.ListingKey, err)
	}
	return err
}

func flushCoordinates(ctx context.Context, tx pgx.Tx, pairs []coordPair) error {
	if len(pairs) == 0 {
		return nil
	}
	for i := 0; i < len(pairs); i += 250 {
		end := i + 250
		if end > len(pairs) {
			end = len(pairs)
		}
		segment := pairs[i:end]
		var parts []string
		var args []any
		args = append(args, "NOW()")
		n := 2
		for _, p := range segment {
			parts = append(parts, fmt.Sprintf("($%d::varchar, $%d::varchar, ST_SetSRID(ST_MakePoint($%d::float8, $%d::float8), 4326)::geography)", n, n+1, n+2, n+3))
			args = append(args, p.datasetSlug, p.listingKey, p.lng, p.lat)
			n += 4
		}
		sql := fmt.Sprintf(`
			UPDATE listings AS l SET coordinates = v.geom, updated_at = $1
			FROM (VALUES %s) AS v(ds, k, geom)
			WHERE l.dataset_slug = v.ds AND l.listing_key = v.k
		`, strings.Join(parts, ","))
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			return err
		}
	}
	return nil
}

func flushNullCoordinates(ctx context.Context, tx pgx.Tx, grouped map[string]map[string]bool) error {
	for datasetSlug, keysMap := range grouped {
		keys := make([]string, 0, len(keysMap))
		for k := range keysMap {
			keys = append(keys, k)
		}
		for i := 0; i < len(keys); i += 250 {
			end := i + 250
			if end > len(keys) {
				end = len(keys)
			}
			segment := keys[i:end]
			_, err := tx.Exec(ctx, `
				UPDATE listings SET coordinates = NULL, updated_at = NOW()
				WHERE dataset_slug = $1 AND listing_key = ANY($2)
			`, datasetSlug, segment)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *ListingMirrorWriter) flushDeletes(ctx context.Context, grouped map[string]map[string]bool) error {
	for datasetSlug, keysMap := range grouped {
		keys := make([]string, 0, len(keysMap))
		for k := range keysMap {
			keys = append(keys, k)
		}
		for i := 0; i < len(keys); i += 250 {
			end := i + 250
			if end > len(keys) {
				end = len(keys)
			}
			segment := keys[i:end]
			var placeholders []string
			var args []any
			n := 1
			for _, key := range segment {
				placeholders = append(placeholders, fmt.Sprintf("($%d::varchar, $%d::varchar)", n, n+1))
				args = append(args, datasetSlug, key)
				n += 2
			}
			sql := fmt.Sprintf(`DELETE FROM listings WHERE (dataset_slug, listing_key) IN (%s)`, strings.Join(placeholders, ","))
			if _, err := w.db.Pool.Exec(ctx, sql, args...); err != nil {
				return err
			}
		}
	}
	return nil
}
