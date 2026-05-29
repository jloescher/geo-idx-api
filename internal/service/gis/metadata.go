package gis

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
	gisrepo "github.com/quantyralabs/idx-api/internal/repository/gis"
)

// MetadataService probes ArcGIS layer metadata for cache invalidation.
type MetadataService struct {
	cfg    config.Config
	db     *repository.DB
	gis    *gisrepo.Repository
	logger *slog.Logger
	http   *http.Client
}

func NewMetadataService(cfg config.Config, db *repository.DB, logger *slog.Logger) *MetadataService {
	timeout := 12 * time.Second
	return &MetadataService{
		cfg:    cfg,
		db:     db,
		gis:    gisrepo.New(db),
		logger: logger,
		http:   &http.Client{Timeout: timeout},
	}
}

// ProbeResult summarizes one source probe attempt.
type ProbeResult struct {
	SourceKey string `json:"source_key"`
	OK        bool   `json:"ok"`
	Error     string `json:"error,omitempty"`
}

// ProbeAllResult is returned to the admin dashboard after probing many sources.
type ProbeAllResult struct {
	OK     bool          `json:"ok"`
	Failed []ProbeResult `json:"failed,omitempty"`
}

// ProbeSource probes one catalog source by key.
func (m *MetadataService) ProbeSource(ctx context.Context, sourceKey string) error {
	if sourceKey == "" {
		result := m.ProbeAll(ctx)
		if len(result.Failed) > 0 {
			return fmt.Errorf("gis probe failed for %d source(s)", len(result.Failed))
		}
		return nil
	}
	endpoint := ""
	switch sourceKey {
	case FDOTAdminBoundariesKey:
		endpoint = "https://gis.fdot.gov/arcgis/rest/services/Admin_Boundaries/FeatureServer/6?f=json"
	case FloridaStatewideCadastralKey:
		endpoint = queryMetaURL(FloridaStatewideCadastralURL)
	default:
		if spec, ok := ParcelSourceSpecByKey(sourceKey); ok {
			endpoint = queryMetaURL(spec.QueryURL)
		} else {
			row, err := m.gis.GetParcelSourceByKey(ctx, sourceKey)
			if err != nil {
				return err
			}
			endpoint = queryMetaURL(row.QueryURL)
		}
	}
	return m.probeOne(ctx, sourceKey, endpoint)
}

func (m *MetadataService) ProbeAll(ctx context.Context) ProbeAllResult {
	endpoints := map[string]string{
		FDOTAdminBoundariesKey:       "https://gis.fdot.gov/arcgis/rest/services/Admin_Boundaries/FeatureServer/6?f=json",
		FloridaStatewideCadastralKey: queryMetaURL(FloridaStatewideCadastralURL),
	}
	for _, spec := range EnabledParcelSourcesForConfig(m.cfg.GIS) {
		if spec.CountySlug == "" || spec.CountySlug == "statewide" {
			continue
		}
		endpoints[spec.SourceKey] = queryMetaURL(spec.QueryURL)
	}
	rows, err := m.gis.ListEnabledParcelSources(ctx)
	if err != nil {
		m.logger.Warn("gis probe list db sources failed", "error", err)
	} else {
		for _, row := range rows {
			if row.SyncMode == SyncModeShapefile {
				continue
			}
			if strings.TrimSpace(row.QueryURL) == "" {
				continue
			}
			endpoints[row.SourceKey] = queryMetaURL(row.QueryURL)
		}
	}
	var failed []ProbeResult
	for key, endpoint := range endpoints {
		if err := m.probeOne(ctx, key, endpoint); err != nil {
			m.logger.Warn("gis probe failed", "source", key, "error", err)
			failed = append(failed, ProbeResult{SourceKey: key, OK: false, Error: err.Error()})
		}
	}
	return ProbeAllResult{OK: len(failed) == 0, Failed: failed}
}

func queryMetaURL(queryURL string) string {
	base := strings.TrimSuffix(strings.TrimSpace(queryURL), "/query")
	return base + "?f=json"
}

func (m *MetadataService) probeOne(ctx context.Context, sourceKey, endpoint string) error {
	if err := m.gis.EnsureSourceState(ctx, sourceKey); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := m.http.Do(req)
	if err != nil {
		_ = m.gis.UpdateProbeResult(ctx, sourceKey, false, 0, err.Error())
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = m.gis.UpdateProbeResult(ctx, sourceKey, false, resp.StatusCode, err.Error())
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := string(body)
		if len(msg) > 200 {
			msg = msg[:200]
		}
		_ = m.gis.UpdateProbeResult(ctx, sourceKey, false, resp.StatusCode, msg)
		return fmt.Errorf("gis probe http %d source=%s", resp.StatusCode, sourceKey)
	}
	_ = m.gis.UpdateProbeResult(ctx, sourceKey, true, resp.StatusCode, "")
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
