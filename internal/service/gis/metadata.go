package gis

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// MetadataService probes ArcGIS layer metadata for cache invalidation.
type MetadataService struct {
	cfg    config.Config
	db     *repository.DB
	logger *slog.Logger
	http   *http.Client
}

func NewMetadataService(cfg config.Config, db *repository.DB, logger *slog.Logger) *MetadataService {
	timeout := 12 * time.Second
	return &MetadataService{
		cfg:    cfg,
		db:     db,
		logger: logger,
		http:   &http.Client{Timeout: timeout},
	}
}

var probeEndpoints = map[string]string{
	"florida_statewide_cadastral": FloridaStatewideCadastralMetaURL,
	"pinellas_enterprise_parcels": PinellasParcelsMetaURL,
	"hillsborough_hc_parcels":     HillsboroughParcelsMetaURL,
	"fdot_admin_boundaries":       "https://gis.fdot.gov/arcgis/rest/services/Admin_Boundaries/FeatureServer/6?f=json",
}

func (m *MetadataService) ProbeAll(ctx context.Context) error {
	for key, endpoint := range probeEndpoints {
		if err := m.probeOne(ctx, key, endpoint); err != nil {
			m.logger.Warn("gis probe failed", "source", key, "error", err)
		}
	}
	return nil
}

func (m *MetadataService) probeOne(ctx context.Context, sourceKey, endpoint string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := m.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fp := metadataFingerprint(body)
	var prev sql.NullString
	err = m.db.Pool.QueryRow(ctx, `SELECT fingerprint FROM gis_source_states WHERE source_key = $1`, sourceKey).Scan(&prev)
	if err != nil {
		return err
	}
	changed := !prev.Valid || prev.String != fp
	if changed {
		_, err = m.db.Pool.Exec(ctx, `
			UPDATE gis_source_states
			SET fingerprint = $2, generation = generation + 1, last_changed_at = NOW(),
			    last_checked_at = NOW(), updated_at = NOW()
			WHERE source_key = $1
		`, sourceKey, fp)
		m.logger.Info("gis source generation bumped", "source", sourceKey)
	} else {
		_, err = m.db.Pool.Exec(ctx, `
			UPDATE gis_source_states SET last_checked_at = NOW(), updated_at = NOW() WHERE source_key = $1
		`, sourceKey)
	}
	return err
}

func metadataFingerprint(body []byte) string {
	var doc map[string]any
	_ = json.Unmarshal(body, &doc)
	parts := []string{
		jsonString(doc["currentVersion"]),
		jsonString(doc["serviceItemId"]),
	}
	if ei, ok := doc["editingInfo"].(map[string]any); ok {
		parts = append(parts, jsonString(ei["lastEditDate"]))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func jsonString(v any) string {
	if v == nil {
		return ""
	}
	b, _ := json.Marshal(v)
	return string(b)
}
