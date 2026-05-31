package gis

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	ogrSrc, ogrCleanup, err := prepareOGRInput(args.StoragePath)
	if err != nil {
		s.failUpload(ctx, args.UploadID, err)
		return err
	}
	defer ogrCleanup()
	s.logger.Info("gis shapefile import prepared ogr input",
		"source", args.SourceKey,
		"upload_id", args.UploadID,
		"ogr_source", ogrSrc,
		"extracted", !strings.HasPrefix(ogrSrc, "/vsizip/"),
	)
	if err := runOGR2OGRGeoJSONSeqWithSource(ctx, ogrSrc, outSeq); err != nil {
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
	src, cleanup, err := prepareOGRInput(inputPath)
	if err != nil {
		return err
	}
	defer cleanup()
	return runOGR2OGRGeoJSONSeqWithSource(ctx, src, outPath)
}

func runOGR2OGRGeoJSONSeqWithSource(ctx context.Context, src, outPath string) error {
	if _, err := exec.LookPath("ogr2ogr"); err != nil {
		return fmt.Errorf("ogr2ogr not installed in worker image: %w", err)
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

// prepareOGRInput returns a filesystem path for ogr2ogr. Zip uploads are extracted to a temp
// directory because GDAL /vsizip/ is unreliable for large DBF reads on Alpine (Polk-scale zips).
func prepareOGRInput(inputPath string) (ogrPath string, cleanup func(), err error) {
	cleanup = func() {}
	if !strings.HasSuffix(strings.ToLower(inputPath), ".zip") {
		return inputPath, cleanup, nil
	}
	inner, err := findShapefileInZip(inputPath)
	if err != nil {
		return "", cleanup, err
	}
	if inner == "" {
		return "/vsizip/" + inputPath, cleanup, nil
	}
	dir, shp, err := extractShapefileSet(inputPath, inner)
	if err != nil {
		return "", cleanup, err
	}
	return shp, func() { _ = os.RemoveAll(dir) }, nil
}

func extractShapefileSet(zipPath, innerShp string) (tmpdir, shpPath string, err error) {
	innerShp = filepath.ToSlash(innerShp)
	dirPrefix := filepath.ToSlash(filepath.Dir(innerShp))
	if dirPrefix == "." {
		dirPrefix = ""
	} else {
		dirPrefix += "/"
	}

	tmpdir, err = os.MkdirTemp("", "gis-shp-import-*")
	if err != nil {
		return "", "", err
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(tmpdir)
		}
	}()

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", "", fmt.Errorf("open zip %s: %w", zipPath, err)
	}
	defer r.Close()

	extracted := 0
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := filepath.ToSlash(f.Name)
		if strings.HasPrefix(name, "__MACOSX/") {
			continue
		}
		if dirPrefix != "" {
			if !strings.HasPrefix(name, dirPrefix) {
				continue
			}
		} else if strings.Contains(name, "/") {
			continue
		}
		dest := filepath.Join(tmpdir, filepath.Base(name))
		if err := extractZipEntry(f, dest); err != nil {
			return "", "", fmt.Errorf("extract %s: %w", name, err)
		}
		extracted++
	}
	if extracted == 0 {
		return "", "", fmt.Errorf("no files extracted for shapefile %s in %s", innerShp, zipPath)
	}

	shpPath = filepath.Join(tmpdir, filepath.Base(innerShp))
	for _, ext := range []string{".shp", ".dbf", ".shx"} {
		side := strings.TrimSuffix(shpPath, ".shp") + ext
		if _, statErr := os.Stat(side); statErr != nil {
			return "", "", fmt.Errorf("shapefile sidecar missing after extract: %s", filepath.Base(side))
		}
	}
	return tmpdir, shpPath, nil
}

func extractZipEntry(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	mode := f.Mode().Perm()
	if mode == 0 {
		mode = 0o640
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, rc); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// shapefileOGRSource returns the ogr2ogr /vsizip/ path (used in tests; production extracts zips to disk).
func shapefileOGRSource(inputPath string) (string, error) {
	if !strings.HasSuffix(strings.ToLower(inputPath), ".zip") {
		return inputPath, nil
	}
	inner, err := findShapefileInZip(inputPath)
	if err != nil {
		return "", err
	}
	if inner != "" {
		return "/vsizip/" + inputPath + "/" + inner, nil
	}
	return "/vsizip/" + inputPath, nil
}

// findShapefileInZip locates a .shp member inside a zip. Prefers the shallowest path,
// then the largest .shp when multiple candidates exist (nested county exports).
func findShapefileInZip(zipPath string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("open zip %s: %w", zipPath, err)
	}
	defer r.Close()

	type candidate struct {
		name  string
		depth int
		size  uint64
	}
	var best *candidate
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := filepath.ToSlash(f.Name)
		if strings.HasPrefix(name, "__MACOSX/") {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(name), ".shp") {
			continue
		}
		depth := strings.Count(strings.Trim(name, "/"), "/")
		c := candidate{name: name, depth: depth, size: f.UncompressedSize64}
		if best == nil || c.depth < best.depth || (c.depth == best.depth && c.size > best.size) {
			best = &c
		}
	}
	if best == nil {
		return "", nil
	}
	return best.name, nil
}

var ErrUploadTooLarge = fmt.Errorf("upload exceeds max size")

// SaveUploadStream writes a multipart upload under GIS_IMPORT_PATH without buffering the full file in memory.
func SaveUploadStream(basePath, sourceKey, filename string, r io.Reader, maxBytes int64) (path string, written int64, err error) {
	dir := filepath.Join(basePath, sourceKey)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", 0, err
	}
	safe := filepath.Base(filename)
	if safe == "" || safe == "." {
		safe = "upload.zip"
	}
	dest := filepath.Join(dir, safe)
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return "", 0, err
	}
	defer func() {
		if err != nil {
			_ = os.Remove(dest)
		}
	}()

	limited := io.Reader(r)
	if maxBytes > 0 {
		limited = io.LimitReader(r, maxBytes+1)
	}
	written, err = io.Copy(f, limited)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if maxBytes > 0 && written > maxBytes {
		return "", written, fmt.Errorf("%w %d bytes", ErrUploadTooLarge, maxBytes)
	}
	if err != nil {
		return "", written, err
	}
	return dest, written, nil
}

// SaveUpload writes multipart upload bytes under GIS_IMPORT_PATH.
func SaveUpload(basePath, sourceKey, filename string, data []byte) (string, error) {
	path, _, err := SaveUploadStream(basePath, sourceKey, filename, bytes.NewReader(data), 0)
	return path, err
}
