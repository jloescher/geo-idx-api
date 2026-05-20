package sync

import (
	"context"
	"encoding/json"

	"github.com/quantyralabs/idx-api/internal/repository"
)

// ListingMirrorWriter upserts rows into listings mirror with PostGIS coordinates.
type ListingMirrorWriter struct {
	db *repository.DB
}

func NewListingMirrorWriter(db *repository.DB) *ListingMirrorWriter {
	return &ListingMirrorWriter{db: db}
}

func (w *ListingMirrorWriter) UpsertBatch(ctx context.Context, dataset string, rows []json.RawMessage) error {
	for _, raw := range rows {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		key, _ := m["ListingKey"].(string)
		if key == "" {
			key, _ = m["ListingId"].(string)
		}
		status, _ := m["StandardStatus"].(string)
		lat, lng := extractCoords(m)
		_, err := w.db.Pool.Exec(ctx, `
			INSERT INTO listings (dataset_slug, listing_key, standard_status, raw_data, latitude, longitude, coordinates, updated_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6,
				CASE WHEN $5 IS NOT NULL AND $6 IS NOT NULL THEN ST_SetSRID(ST_MakePoint($6, $5), 4326)::geography ELSE NULL END,
				NOW(), NOW())
			ON CONFLICT (dataset_slug, listing_key) DO UPDATE SET
				standard_status = EXCLUDED.standard_status,
				raw_data = EXCLUDED.raw_data,
				latitude = EXCLUDED.latitude,
				longitude = EXCLUDED.longitude,
				coordinates = EXCLUDED.coordinates,
				updated_at = NOW()
		`, dataset, key, status, raw, lat, lng)
		if err != nil {
			return err
		}
	}
	return nil
}

func extractCoords(m map[string]any) (*float64, *float64) {
	if lat, ok := m["Latitude"].(float64); ok {
		if lng, ok2 := m["Longitude"].(float64); ok2 {
			return &lat, &lng
		}
	}
	return nil, nil
}
