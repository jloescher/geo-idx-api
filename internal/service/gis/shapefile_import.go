package gis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/quantyralabs/idx-api/internal/config"
	gisrepo "github.com/quantyralabs/idx-api/internal/repository/gis"
)

// ShapefileImportArgs is the payload for gis.shapefile_import jobs.
type ShapefileImportArgs struct {
	SourceKey   string `json:"source_key"`
	StoragePath string `json:"storage_path"`
	UploadID    int64  `json:"upload_id,omitempty"`
}

// ShapefileImportService ingests uploaded parcel geometry into gis_parcels.
type ShapefileImportService struct {
	cfg    config.Config
	db     *gisrepo.Repository
	logger *slog.Logger
}

func NewShapefileImportService(cfg config.Config, repo *gisrepo.Repository, logger *slog.Logger) *ShapefileImportService {
	return &ShapefileImportService{cfg: cfg, db: repo, logger: logger}
}

// Import reads a shapefile or zip on disk via ogr2ogr → GeoJSON, then upserts parcels.
func (s *ShapefileImportService) Import(ctx context.Context, args ShapefileImportArgs) error {
	if args.SourceKey == "" || args.StoragePath == "" {
		return fmt.Errorf("source_key and storage_path required")
	}
	spec, err := NewParcelSyncService(s.cfg, s.db, nil, s.logger).resolveParcelSource(ctx, args.SourceKey)
	if err != nil {
		return err
	}
	if err := s.db.EnsureSourceState(ctx, args.SourceKey); err != nil {
		return err
	}
	gen, err := s.db.BumpSourceGeneration(ctx, args.SourceKey)
	if err != nil {
		return err
	}
	outJSON := args.StoragePath + ".geojson"
	if err := runOGR2OGR(ctx, args.StoragePath, outJSON); err != nil {
		return err
	}
	defer os.Remove(outJSON)

	body, err := os.ReadFile(outJSON)
	if err != nil {
		return err
	}
	page, err := ParseFeatureCollection(body)
	if err != nil {
		return err
	}
	chunk := s.cfg.GIS.SyncUpsertChunk
	if chunk <= 0 {
		chunk = 500
	}
	var batch []gisrepo.ParcelRow
	fpStr := ""
	statewide := IsStatewideCadastralSource(args.SourceKey)
	idFields := spec.ParcelIDFields
	for _, feat := range page.Features {
		county := spec.CountySlug
		if statewide {
			coNo, ok := CONOFromProperties(feat.Properties)
			if !ok {
				continue
			}
			slug, ok := CountySlugFromCONO(coNo)
			if !ok || !IsMLSPilotCounty(slug) {
				continue
			}
			county = slug
		}
		row, err := ExtractParcelRow(feat, args.SourceKey, county, int(gen), &fpStr, idFields)
		if err != nil {
			continue
		}
		batch = append(batch, row)
		if len(batch) >= chunk {
			if err := s.db.BulkUpsertParcels(ctx, batch, chunk); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		if err := s.db.BulkUpsertParcels(ctx, batch, chunk); err != nil {
			return err
		}
	}
	staleCounty := spec.CountySlug
	if statewide {
		staleCounty = ""
	}
	if _, err := s.db.DeleteStaleParcels(ctx, args.SourceKey, staleCounty, int(gen)); err != nil {
		return err
	}
	_ = s.db.TouchParcelSourceSynced(ctx, args.SourceKey)
	s.logger.Info("gis shapefile import done", "source", args.SourceKey, "features", len(page.Features))
	return nil
}

func runOGR2OGR(ctx context.Context, inputPath, outPath string) error {
	if _, err := exec.LookPath("ogr2ogr"); err != nil {
		return fmt.Errorf("ogr2ogr not installed in worker image: %w", err)
	}
	src := inputPath
	if strings.HasSuffix(strings.ToLower(inputPath), ".zip") {
		src = "/vsizip/" + inputPath
	}
	cmd := exec.CommandContext(ctx, "ogr2ogr", "-f", "GeoJSON", outPath, src, "-t_srs", "EPSG:4326")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ogr2ogr: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// UnmarshalShapefileImportArgs parses job payload args.
func UnmarshalShapefileImportArgs(raw json.RawMessage) (ShapefileImportArgs, error) {
	var args ShapefileImportArgs
	err := json.Unmarshal(raw, &args)
	return args, err
}

// SaveUpload writes multipart upload bytes under GIS_IMPORT_PATH.
func SaveUpload(basePath, sourceKey, filename string, data []byte) (string, error) {
	dir := filepath.Join(basePath, sourceKey)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	safe := filepath.Base(filename)
	if safe == "" || safe == "." {
		safe = "upload.zip"
	}
	dest := filepath.Join(dir, safe)
	if err := os.WriteFile(dest, data, 0o640); err != nil {
		return "", err
	}
	return dest, nil
}
