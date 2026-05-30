package gis

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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

// Import reads a shapefile or zip on disk via ogr2ogr → GeoJSONSeq, streaming features into gis_parcels.
func (s *ShapefileImportService) Import(ctx context.Context, args ShapefileImportArgs) error {
	if args.SourceKey == "" || args.StoragePath == "" {
		return fmt.Errorf("source_key and storage_path required")
	}
	started := time.Now()
	s.logger.Info("gis shapefile import started",
		"source", args.SourceKey,
		"upload_id", args.UploadID,
		"path", args.StoragePath,
	)

	if args.UploadID > 0 {
		_ = s.db.SetImportUploadStatus(ctx, args.UploadID, "processing", "")
	}
	spec, err := NewParcelSyncService(s.cfg, s.db, nil, s.logger).resolveParcelSource(ctx, args.SourceKey)
	if err != nil {
		s.failUpload(ctx, args.UploadID, err)
		return err
	}
	if err := s.db.EnsureSourceState(ctx, args.SourceKey); err != nil {
		return err
	}
	if err := ensureImportFileReadable(args.StoragePath); err != nil {
		s.failUpload(ctx, args.UploadID, err)
		return err
	}

	outSeq := args.StoragePath + ".geojsonl"
	ogrStarted := time.Now()
	if err := runOGR2OGRGeoJSONSeq(ctx, args.StoragePath, outSeq); err != nil {
		s.failUpload(ctx, args.UploadID, err)
		return err
	}
	defer os.Remove(outSeq)
	s.logger.Info("gis shapefile import ogr2ogr done",
		"source", args.SourceKey,
		"upload_id", args.UploadID,
		"elapsed_sec", int(time.Since(ogrStarted).Seconds()),
	)

	gen, err := s.db.BumpSourceGeneration(ctx, args.SourceKey)
	if err != nil {
		s.failUpload(ctx, args.UploadID, err)
		return err
	}

	chunk := s.cfg.GIS.SyncUpsertChunk
	if chunk <= 0 {
		chunk = 500
	}
	statewide := IsStatewideCadastralSource(args.SourceKey)
	idFields := spec.ParcelIDFields
	fpStr := ""
	var batch batchState
	batch.chunk = chunk
	featureCount, err := streamGeoJSONSeqFeatures(outSeq, func(feat ArcGISFeature) error {
		county := spec.CountySlug
		if statewide {
			coNo, ok := CONOFromProperties(feat.Properties)
			if !ok {
				return nil
			}
			slug, ok := CountySlugFromCONO(coNo)
			if !ok || !IsMLSPilotCounty(slug) {
				return nil
			}
			county = slug
		}
		row, err := ExtractParcelRow(feat, args.SourceKey, county, int(gen), &fpStr, idFields)
		if err != nil {
			return nil
		}
		return upsertParcelBatch(ctx, s.db, &batch, row)
	})
	if err != nil {
		s.failUpload(ctx, args.UploadID, err)
		return err
	}
	if err := flushParcelBatch(ctx, s.db, batch); err != nil {
		s.failUpload(ctx, args.UploadID, err)
		return err
	}

	staleCounty := spec.CountySlug
	if statewide {
		staleCounty = ""
	}
	if _, err := s.db.DeleteStaleParcels(ctx, args.SourceKey, staleCounty, int(gen)); err != nil {
		s.failUpload(ctx, args.UploadID, err)
		return err
	}
	_ = s.db.TouchParcelSourceSynced(ctx, args.SourceKey)
	_ = s.db.SetImportUploadStatus(ctx, args.UploadID, "done", "")
	_ = s.db.UpdateProbeResult(ctx, args.SourceKey, true, 0, "")
	s.logger.Info("gis shapefile import done",
		"source", args.SourceKey,
		"upload_id", args.UploadID,
		"features", featureCount,
		"generation", gen,
		"elapsed_sec", int(time.Since(started).Seconds()),
	)
	return nil
}

type batchState struct {
	chunk int
	rows  []gisrepo.ParcelRow
}

func upsertParcelBatch(ctx context.Context, db *gisrepo.Repository, st *batchState, row gisrepo.ParcelRow) error {
	st.rows = append(st.rows, row)
	if len(st.rows) < st.chunk {
		return nil
	}
	if err := db.BulkUpsertParcels(ctx, st.rows, st.chunk); err != nil {
		return err
	}
	st.rows = st.rows[:0]
	return nil
}

func flushParcelBatch(ctx context.Context, db *gisrepo.Repository, st batchState) error {
	if len(st.rows) == 0 {
		return nil
	}
	return db.BulkUpsertParcels(ctx, st.rows, st.chunk)
}

func streamGeoJSONSeqFeatures(path string, fn func(ArcGISFeature) error) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 256*1024)
	scanner.Buffer(buf, 16*1024*1024)

	count := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		feat, err := ParseGeoJSONFeature([]byte(line))
		if err != nil {
			var collection map[string]any
			if json.Unmarshal([]byte(line), &collection) == nil {
				if typ, _ := collection["type"].(string); typ == "FeatureCollection" {
					continue
				}
			}
			return count, fmt.Errorf("parse feature line %d: %w", count+1, err)
		}
		if err := fn(feat); err != nil {
			return count, err
		}
		count++
	}
	if err := scanner.Err(); err != nil {
		return count, err
	}
	if count == 0 {
		return 0, fmt.Errorf("ogr2ogr produced no features in %s", path)
	}
	return count, nil
}

func (s *ShapefileImportService) failUpload(ctx context.Context, uploadID int64, err error) {
	if uploadID <= 0 || err == nil {
		return
	}
	_ = s.db.SetImportUploadStatus(ctx, uploadID, "failed", err.Error())
}

func ensureImportFileReadable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("upload file not found at %s: mount the same GIS_IMPORT_PATH volume on idx-api-web and idx-api-worker on the host that received the upload (multi-DC: use shared storage or a single upload origin)", path)
		}
		return fmt.Errorf("upload file stat %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("upload path is a directory, expected file: %s", path)
	}
	return nil
}

func runOGR2OGRGeoJSONSeq(ctx context.Context, inputPath, outPath string) error {
	if _, err := exec.LookPath("ogr2ogr"); err != nil {
		return fmt.Errorf("ogr2ogr not installed in worker image: %w", err)
	}
	src := inputPath
	if strings.HasSuffix(strings.ToLower(inputPath), ".zip") {
		src = "/vsizip/" + inputPath
	}
	cmd := exec.CommandContext(ctx, "ogr2ogr", "-f", "GeoJSONSeq", outPath, src, "-t_srs", "EPSG:4326", "-dim", "2")
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
