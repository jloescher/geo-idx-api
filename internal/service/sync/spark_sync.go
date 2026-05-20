package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
)

func activePendingReplicationBaseFilter(cfg config.Config) string {
	cutoff := MirrorRollingCutoff(cfg)
	if cutoff == nil {
		return "StandardStatus eq 'Active' or StandardStatus eq 'Pending'"
	}
	return fmt.Sprintf("(StandardStatus eq 'Active' or StandardStatus eq 'Pending') and ModificationTimestamp gt %s",
		sparkRollingTimestampLiteral(*cutoff))
}

// SparkSync fetches Spark replication OData pages (replication.sparkapi.com only).
type SparkSync struct {
	cfg     config.Config
	http    *http.Client
	cursors *CursorStore
}

func NewSparkSync(cfg config.Config, db *repository.DB) *SparkSync {
	return &SparkSync{
		cfg: cfg,
		http: &http.Client{
			Timeout: cfg.Spark.Timeout,
		},
		cursors: NewCursorStore(db),
	}
}

func (s *SparkSync) FetchReplicationPage(ctx context.Context, cursor SyncCursor) (PageResult, error) {
	top := s.cfg.Spark.SyncReplicationTop
	if top <= 0 {
		top = 1000
	}
	if top > 1000 {
		top = 1000
	}

	var fetchURL string
	query := url.Values{}

	if cursor.ReplicationNextURL != nil && *cursor.ReplicationNextURL != "" {
		fetchURL = *cursor.ReplicationNextURL
	} else {
		fetchURL = s.propertyCollectionURL()
		query.Set("$filter", SparkReplicationFilter(s.cfg))
		query.Set("$top", fmt.Sprintf("%d", top))
		s.applySyncExpand(query)
	}

	return s.fetchPage(ctx, fetchURL, query, cursor.DatasetSlug, true)
}

func (s *SparkSync) FetchIncrementalPage(ctx context.Context, cursor SyncCursor, skip int) (PageResult, error) {
	if cursor.LastModificationTimestamp == nil {
		return PageResult{ReplicationComplete: true}, nil
	}

	top := s.cfg.Spark.SyncIncrementalTop
	if top <= 0 {
		top = 1000
	}
	if top > 1000 {
		top = 1000
	}

	windowEnd := cursor.IncrementalWindowEnd
	if windowEnd == nil {
		now := time.Now().UTC()
		windowEnd = &now
	}

	lower := cursor.LastModificationTimestamp.UTC().Format("2006-01-02T15:04:05Z")
	upper := windowEnd.UTC().Format("2006-01-02T15:04:05Z")
	filter := fmt.Sprintf("(%s) and ModificationTimestamp gt %s and ModificationTimestamp lt %s",
		activePendingReplicationBaseFilter(s.cfg), lower, upper)

	fetchURL := s.propertyCollectionURL()
	query := url.Values{}
	query.Set("$filter", filter)
	query.Set("$orderby", "ModificationTimestamp asc")
	query.Set("$top", fmt.Sprintf("%d", top))
	query.Set("$skip", fmt.Sprintf("%d", skip))
	s.applySyncExpand(query)

	result, err := s.fetchPage(ctx, fetchURL, query, cursor.DatasetSlug, false)
	if err != nil {
		return result, err
	}
	if cursor.IncrementalWindowEnd == nil {
		result.IncrementalWindowEnd = windowEnd
	}
	return result, nil
}

func (s *SparkSync) applySyncExpand(query url.Values) {
	if expand := strings.TrimSpace(s.cfg.MLS.SyncExpand); expand != "" {
		query.Set("$expand", expand)
	}
}

func (s *SparkSync) fetchPage(ctx context.Context, fetchURL string, query url.Values, dataset string, replication bool) (PageResult, error) {
	if s.cfg.Spark.AccessToken == "" {
		return PageResult{}, fmt.Errorf("SPARK_ACCESS_TOKEN is not configured")
	}

	reqURL := fetchURL
	if len(query) > 0 {
		sep := "?"
		if strings.Contains(fetchURL, "?") {
			sep = "&"
		}
		reqURL = fetchURL + sep + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return PageResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.Spark.AccessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return PageResult{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return PageResult{}, err
	}

	result := PageResult{
		FetchURL:    fetchURL,
		UpstreamURL: reqURL,
		ODataQuery:  odataQueryMap(query),
		HTTPStatus:  resp.StatusCode,
	}

	if resp.StatusCode == 403 {
		result.Forbidden = true
		return result, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.HTTPError = true
		return result, nil
	}

	var parsed struct {
		Value []json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return PageResult{}, fmt.Errorf("decode spark odata: %w", err)
	}
	rows := parsed.Value
	if rows == nil {
		rows = []json.RawMessage{}
	}

	result.Rows = rows
	result.MaxModificationTs = maxModificationTimestampFromRows(dataset, rows)
	next := extractNextURL(resp.Header.Get("Link"), body)
	result.NextReplicationURL = next

	if replication {
		result.ReplicationComplete = next == nil || *next == ""
		return result, nil
	}

	top := s.cfg.Spark.SyncIncrementalTop
	if top <= 0 {
		top = 1000
	}
	if len(rows) == 0 {
		result.ReplicationComplete = true
		return result, nil
	}
	result.IncrementalHasMore = len(rows) >= top
	return result, nil
}

func (s *SparkSync) propertyCollectionURL() string {
	host := strings.TrimRight(s.cfg.Spark.ReplicationHost, "/")
	root := strings.Trim(s.cfg.Spark.ReplicationReso, "/")
	return host + "/" + root + "/Property"
}
