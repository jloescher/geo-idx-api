package admin

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/queue"
	"github.com/quantyralabs/idx-api/internal/repository"
	gisrepo "github.com/quantyralabs/idx-api/internal/repository/gis"
	"github.com/quantyralabs/idx-api/internal/service/gis"
)

// GISHandler exposes operator GIS catalog and sync controls.
type GISHandler struct {
	cfg    config.Config
	db     *repository.DB
	logger *slog.Logger
}

// NewGISHandler constructs the admin GIS handler.
func NewGISHandler(cfg config.Config, db *repository.DB, logger *slog.Logger) *GISHandler {
	return &GISHandler{cfg: cfg, db: db, logger: logger}
}

type gisProbeRequest struct {
	SourceKey string `json:"source_key"`
}

type gisSyncRequest struct {
	SourceKey string `json:"source_key"`
	Force     bool   `json:"force"`
}

type parcelSourcePayload struct {
	SourceKey      string   `json:"source_key"`
	CountySlug     string   `json:"county_slug"`
	QueryURL       string   `json:"query_url"`
	SyncMode       string   `json:"sync_mode"`
	ArcGISWhere    *string  `json:"arcgis_where"`
	BBoxWest       *float64 `json:"bbox_west"`
	BBoxSouth      *float64 `json:"bbox_south"`
	BBoxEast       *float64 `json:"bbox_east"`
	BBoxNorth      *float64 `json:"bbox_north"`
	HTTPTimeoutSec *int     `json:"http_timeout_sec"`
	PageSize       *int     `json:"page_size"`
	MLSFeed        string   `json:"mls_feed"`
	Enabled        *bool    `json:"enabled"`
	Priority       *int     `json:"priority"`
	Notes          *string  `json:"notes"`
}

func (h *GISHandler) requireAdmin(c *fiber.Ctx) error {
	uid, _ := c.Locals("user_id").(int64)
	var isAdmin bool
	err := h.db.Pool.QueryRow(c.Context(), `SELECT is_admin FROM users WHERE id = $1`, uid).Scan(&isAdmin)
	if err != nil || !isAdmin {
		return fiber.NewError(fiber.StatusForbidden, "admin only")
	}
	return nil
}

// Probe runs metadata probes (one source or all).
func (h *GISHandler) Probe(c *fiber.Ctx) error {
	if err := h.requireAdmin(c); err != nil {
		// #region agent log
		agentDebugLog("GIS-A", "gis.go:Probe", "requireAdmin failed", map[string]any{"error": err.Error()})
		// #endregion
		return err
	}
	var req gisProbeRequest
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON body"})
		}
	}
	// #region agent log
	agentDebugLog("GIS-A", "gis.go:Probe", "handler reached", map[string]any{"source_key": req.SourceKey})
	// #endregion
	meta := gis.NewMetadataService(h.cfg, h.db, h.logger)
	if req.SourceKey == "" {
		result := meta.ProbeAll(c.Context())
		resp := fiber.Map{"ok": result.OK, "scope": "all"}
		if len(result.Failed) > 0 {
			resp["failed"] = result.Failed
		}
		return c.JSON(resp)
	}
	if err := meta.ProbeSource(c.Context(), req.SourceKey); err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true, "source_key": req.SourceKey})
}

// Sync enqueues a parcel sync session for one source.
func (h *GISHandler) Sync(c *fiber.Ctx) error {
	if err := h.requireAdmin(c); err != nil {
		return err
	}
	var req gisSyncRequest
	if err := c.BodyParser(&req); err != nil || req.SourceKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "source_key required"})
	}
	q := queue.NewClient(h.db.Pool, h.cfg.Queue.Table, h.cfg.Queue.NotifyChannel, h.cfg.Queue.RetryAfter, h.cfg.Queue.ReservationTimeout)
	syncSvc := gis.NewParcelSyncService(h.cfg, gisrepo.New(h.db), q, h.logger)
	if err := syncSvc.KickoffSource(c.Context(), req.SourceKey, req.Force); err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"enqueued": true, "source_key": req.SourceKey})
}

// ListSources returns catalog rows joined with source state health.
func (h *GISHandler) ListSources(c *fiber.Ctx) error {
	if err := h.requireAdmin(c); err != nil {
		return err
	}
	repo := gisrepo.New(h.db)
	catalog, err := repo.ListAllParcelSources(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	states, err := repo.ListSourceStates(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	byKey := map[string]gisrepo.SourceStateRow{}
	for _, s := range states {
		byKey[s.SourceKey] = s
	}
	type row struct {
		gisrepo.ParcelSourceRow
		State gisrepo.SourceStateRow `json:"state,omitempty"`
	}
	out := make([]row, 0, len(catalog))
	for _, p := range catalog {
		r := row{ParcelSourceRow: p}
		if st, ok := byKey[p.SourceKey]; ok {
			r.State = st
		}
		out = append(out, r)
	}
	return c.JSON(fiber.Map{"sources": out})
}

// CreateSource upserts a parcel catalog row.
func (h *GISHandler) CreateSource(c *fiber.Ctx) error {
	if err := h.requireAdmin(c); err != nil {
		return err
	}
	var req parcelSourcePayload
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON body"})
	}
	row, err := payloadToParcelRow(req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	repo := gisrepo.New(h.db)
	if err := repo.EnsureParcelSourceCatalog(c.Context(), []gisrepo.ParcelSourceRow{row}); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	_ = repo.EnsureSourceState(c.Context(), row.SourceKey)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"source_key": row.SourceKey})
}

// UpdateSource updates an existing catalog row.
func (h *GISHandler) UpdateSource(c *fiber.Ctx) error {
	if err := h.requireAdmin(c); err != nil {
		return err
	}
	sourceKey := c.Params("source_key")
	var req parcelSourcePayload
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON body"})
	}
	req.SourceKey = sourceKey
	row, err := payloadToParcelRow(req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	repo := gisrepo.New(h.db)
	if err := repo.EnsureParcelSourceCatalog(c.Context(), []gisrepo.ParcelSourceRow{row}); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"source_key": row.SourceKey})
}

// DeleteSource soft-disables or hard-deletes a catalog row.
func (h *GISHandler) DeleteSource(c *fiber.Ctx) error {
	if err := h.requireAdmin(c); err != nil {
		return err
	}
	sourceKey := c.Params("source_key")
	repo := gisrepo.New(h.db)
	if c.Query("hard") == "true" {
		if err := repo.DeleteParcelSource(c.Context(), sourceKey); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"deleted": true, "source_key": sourceKey})
	}
	if err := repo.SetParcelSourceEnabled(c.Context(), sourceKey, false); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"enabled": false, "source_key": sourceKey})
}

// UploadShapefile accepts multipart upload and enqueues shapefile import.
func (h *GISHandler) UploadShapefile(c *fiber.Ctx) error {
	if err := h.requireAdmin(c); err != nil {
		return err
	}
	sourceKey := c.Params("source_key")
	file, err := c.FormFile("file")
	if err != nil {
		// #region agent log
		agentDebugLog("SHP-A", "gis.go:UploadShapefile", "FormFile failed", map[string]any{
			"source_key": sourceKey, "error": err.Error(),
		})
		// #endregion
		if strings.Contains(strings.ToLower(err.Error()), "limit") || strings.Contains(strings.ToLower(err.Error()), "large") {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
				"error": "upload exceeds server body limit; zip parcel shapefiles and ensure API BodyLimit matches GIS_IMPORT_MAX_BYTES",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file required"})
	}
	// #region agent log
	agentDebugLog("SHP-A", "gis.go:UploadShapefile", "upload received", map[string]any{
		"source_key": sourceKey, "filename": file.Filename, "size": file.Size,
	})
	// #endregion
	if h.cfg.GIS.ImportMaxBytes > 0 && file.Size > h.cfg.GIS.ImportMaxBytes {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "file too large"})
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".zip" && ext != ".shp" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "allowed extensions: .zip, .shp"})
	}
	f, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read upload"})
	}
	defer f.Close()
	data := make([]byte, file.Size)
	if _, err := f.Read(data); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read upload"})
	}
	path, err := gis.SaveUpload(h.cfg.GIS.ImportPath, sourceKey, file.Filename, data)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	uid, _ := c.Locals("user_id").(int64)
	var uploadID int64
	if err := h.db.Pool.QueryRow(c.Context(), `
		INSERT INTO gis_import_uploads (source_key, original_filename, storage_path, status, uploaded_by_user_id)
		VALUES ($1, $2, $3, 'pending', NULLIF($4, 0))
		RETURNING id
	`, sourceKey, file.Filename, path, uid).Scan(&uploadID); err != nil {
		h.logger.Error("gis import upload record", "error", err, "source_key", sourceKey)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to record upload"})
	}

	q := queue.NewClient(h.db.Pool, h.cfg.Queue.Table, h.cfg.Queue.NotifyChannel, h.cfg.Queue.RetryAfter, h.cfg.Queue.ReservationTimeout)
	queueName := h.cfg.GIS.SyncQueue
	if queueName == "" {
		queueName = "default"
	}
	jobID, err := q.Enqueue(c.Context(), queueName, queue.TypeGISShapefileImport, gis.ShapefileImportArgs{
		SourceKey:   sourceKey,
		StoragePath: path,
		UploadID:    uploadID,
	}, 0)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	// #region agent log
	agentDebugLog("SHP-B", "gis.go:UploadShapefile", "job enqueued", map[string]any{
		"source_key": sourceKey, "job_id": jobID, "upload_id": uploadID, "queue": queueName, "path": path,
	})
	// #endregion
	return c.JSON(fiber.Map{"job_id": jobID, "upload_id": uploadID, "storage_path": path})
}

func payloadToParcelRow(req parcelSourcePayload) (gisrepo.ParcelSourceRow, error) {
	if req.SourceKey == "" || req.CountySlug == "" {
		return gisrepo.ParcelSourceRow{}, fmt.Errorf("source_key, county_slug required")
	}
	mode := req.SyncMode
	if mode == "" {
		mode = gis.SyncModeBBox
	}
	switch mode {
	case gis.SyncModeBBox, gis.SyncModePaginate, gis.SyncModeWhereFilter, gis.SyncModeShapefile:
	default:
		return gisrepo.ParcelSourceRow{}, fmt.Errorf("invalid sync_mode")
	}
	queryURL := strings.TrimSpace(req.QueryURL)
	if queryURL == "" && mode != gis.SyncModeShapefile {
		return gisrepo.ParcelSourceRow{}, fmt.Errorf("query_url required unless sync_mode=shapefile")
	}
	if queryURL == "" {
		queryURL = "shapefile://local"
	}
	feed := req.MLSFeed
	if feed == "" {
		feed = "stellar"
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	priority := 100
	if req.Priority != nil {
		priority = *req.Priority
	}
	return gisrepo.ParcelSourceRow{
		SourceKey:      req.SourceKey,
		CountySlug:     req.CountySlug,
		QueryURL:       queryURL,
		SyncMode:       mode,
		ArcGISWhere:    req.ArcGISWhere,
		BBoxWest:       req.BBoxWest,
		BBoxSouth:      req.BBoxSouth,
		BBoxEast:       req.BBoxEast,
		BBoxNorth:      req.BBoxNorth,
		HTTPTimeoutSec: req.HTTPTimeoutSec,
		PageSize:       req.PageSize,
		MLSFeed:        feed,
		Enabled:        enabled,
		Priority:       priority,
		Notes:          req.Notes,
	}, nil
}

// RegisterGISRoutes mounts admin GIS operator endpoints (session auth required on parent group).
func RegisterGISRoutes(grp fiber.Router, h *GISHandler, uploadCORS fiber.Handler) {
	grp.Post("/probe", h.Probe)
	grp.Post("/sync", h.Sync)
	grp.Get("/sources", h.ListSources)
	grp.Post("/sources", h.CreateSource)
	grp.Put("/sources/:source_key", h.UpdateSource)
	grp.Delete("/sources/:source_key", h.DeleteSource)
	if uploadCORS != nil {
		grp.Options("/sources/:source_key/upload", uploadCORS)
		grp.Post("/sources/:source_key/upload", uploadCORS, h.UploadShapefile)
		return
	}
	grp.Post("/sources/:source_key/upload", h.UploadShapefile)
}
