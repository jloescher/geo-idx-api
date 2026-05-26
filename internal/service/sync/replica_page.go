package sync

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// ReplicaPageMeta is observability metadata for a staged MLS OData page.
type ReplicaPageMeta struct {
	FetchURL    string
	UpstreamURL string
	ODataQuery  map[string]string
}

// ReplicaPageStore stages gzip MLS pages before chunked persist.
// Revenue impact: splitting large API pages bounds worker memory during replication.
type ReplicaPageStore struct {
	db  *repository.DB
	cfg config.Config
}

func NewReplicaPageStore(db *repository.DB, cfg config.Config) *ReplicaPageStore {
	return &ReplicaPageStore{db: db, cfg: cfg}
}

type pagePayloadV2 struct {
	V     int      `json:"v"`
	Parts []string `json:"parts"`
}

func odataQueryMap(q url.Values) map[string]string {
	if len(q) == 0 {
		return nil
	}
	out := make(map[string]string, len(q))
	for k, vs := range q {
		if len(vs) > 0 {
			out[k] = vs[0]
		}
	}
	return out
}

func (s *ReplicaPageStore) StorePage(
	ctx context.Context,
	provider, dataset, mode string,
	rows []json.RawMessage,
	chunkSize int,
	meta ReplicaPageMeta,
) (int64, int, error) {
	if chunkSize <= 0 {
		chunkSize = 50
	}
	var parts []string
	for i := 0; i < len(rows); i += chunkSize {
		end := i + chunkSize
		if end > len(rows) {
			end = len(rows)
		}
		chunk := rows[i:end]
		b, err := json.Marshal(chunk)
		if err != nil {
			return 0, 0, err
		}
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, _ = zw.Write(b)
		_ = zw.Close()
		parts = append(parts, base64.StdEncoding.EncodeToString(buf.Bytes()))
	}
	payload, _ := json.Marshal(pagePayloadV2{V: 2, Parts: parts})
	var odataJSON []byte
	if len(meta.ODataQuery) > 0 {
		odataJSON, _ = json.Marshal(meta.ODataQuery)
	}
	var fetchURL, upstreamURL *string
	if meta.FetchURL != "" {
		fetchURL = &meta.FetchURL
	}
	if meta.UpstreamURL != "" {
		upstreamURL = &meta.UpstreamURL
	}
	var id int64
	err := s.db.Pool.QueryRow(ctx, `
		INSERT INTO replica_pages (
			provider, dataset_slug, mode, status, compressed_payload, row_count,
			fetch_url, upstream_url, odata_query, fetched_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, 'pending', $4, $5, $6, $7, $8, NOW(), NOW(), NOW())
		RETURNING id
	`, provider, dataset, mode, string(payload), len(rows), fetchURL, upstreamURL, odataJSON).Scan(&id)
	return id, len(parts), err
}

func (s *ReplicaPageStore) MarkProcessing(ctx context.Context, pageID int64, batchID string) error {
	var batch any
	if batchID != "" {
		batch = batchID
	}
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE replica_pages
		SET status = 'processing', batch_id = $2, updated_at = NOW()
		WHERE id = $1
	`, pageID, batch)
	return err
}

func (s *ReplicaPageStore) RowsForChunk(ctx context.Context, pageID int64, chunkIndex, chunkTotal int) ([]json.RawMessage, error) {
	var payload string
	err := s.db.Pool.QueryRow(ctx, `SELECT compressed_payload FROM replica_pages WHERE id = $1`, pageID).Scan(&payload)
	if err != nil {
		return nil, err
	}
	var p pagePayloadV2
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return nil, err
	}
	if chunkIndex < 1 || chunkIndex > len(p.Parts) {
		return nil, fmt.Errorf("chunk index out of range")
	}
	raw, err := base64.StdEncoding.DecodeString(p.Parts[chunkIndex-1])
	if err != nil {
		return nil, err
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	var rows []json.RawMessage
	if err := json.NewDecoder(zr).Decode(&rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *ReplicaPageStore) MarkCompleted(ctx context.Context, pageID int64) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE replica_pages SET status = 'completed', processed_at = NOW(), updated_at = NOW() WHERE id = $1
	`, pageID)
	return err
}

func (s *ReplicaPageStore) DeletePage(ctx context.Context, pageID int64) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM replica_pages WHERE id = $1`, pageID)
	return err
}

func (s *ReplicaPageStore) PurgeEligible(ctx context.Context) error {
	hours := s.cfg.MLS.ReplicaPageRetentionHours
	if hours <= 0 {
		hours = 24
	}
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	_, err := s.db.Pool.Exec(ctx, `
		DELETE FROM replica_pages
		WHERE status = 'completed' AND processed_at < $1
	`, cutoff)
	if err != nil {
		return err
	}
	failedDays := s.cfg.MLS.ReplicaPageFailedRetentionDays
	if failedDays <= 0 {
		failedDays = 7
	}
	failedCutoff := time.Now().AddDate(0, 0, -failedDays)
	_, err = s.db.Pool.Exec(ctx, `
		DELETE FROM replica_pages
		WHERE status = 'failed' AND created_at < $1
	`, failedCutoff)
	return err
}

func (s *ReplicaPageStore) HasActivePage(ctx context.Context, provider, dataset string) (bool, error) {
	var n int
	err := s.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM replica_pages
		WHERE provider = $1 AND dataset_slug = $2 AND status IN ('pending', 'processing')
	`, provider, dataset).Scan(&n)
	return n > 0, err
}

func NewBatchID() uuid.UUID { return uuid.New() }
