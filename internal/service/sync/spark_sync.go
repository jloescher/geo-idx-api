package sync

import (
	"context"
	"encoding/json"
	"fmt"
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
	limiter *ClusterRateLimiter
}

func NewSparkSync(cfg config.Config, db *repository.DB) *SparkSync {
	var limiter *ClusterRateLimiter
	if db != nil {
		limiter = NewSparkClusterRateLimiter(db.Pool, cfg)
	}
	return &SparkSync{
		cfg: cfg,
		http: &http.Client{
			Timeout: cfg.Spark.Timeout,
		},
		cursors: NewCursorStore(db),
		limiter: limiter,
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
		s.applySyncExpand(query, true)
	}

	return s.fetchPage(ctx, fetchURL, query, cursor.DatasetSlug, true)
}

func (s *SparkSync) FetchIncrementalPage(ctx context.Context, cursor SyncCursor, skip int) (PageResult, error) {
	if cursor.LastModificationTimestamp == nil {
		return PageResult{ReplicationComplete: true}, nil
	}

	windowEnd := cursor.IncrementalWindowEnd
	if windowEnd == nil {
		now := time.Now().UTC()
		windowEnd = &now
	}

	// Spark rejects empty windows (gt >= lt). Common right after replication when the
	// cursor already caught up to "now".
	if !cursor.LastModificationTimestamp.Before(*windowEnd) {
		return PageResult{
			ReplicationComplete: true,
			IncrementalWindowEnd: windowEnd,
		}, nil
	}

	result, err := s.fetchIncrementalOnce(ctx, cursor, skip, *windowEnd, true)
	if err != nil {
		return result, err
	}
	if result.HTTPError && result.HTTPStatus == http.StatusBadRequest {
		retry, retryErr := s.fetchIncrementalOnce(ctx, cursor, skip, *windowEnd, false)
		if retryErr != nil {
			return result, retryErr
		}
		if !retry.HTTPError {
			result = retry
		} else {
			result.ODataError = retry.ODataError
		}
	}
	if cursor.IncrementalWindowEnd == nil {
		result.IncrementalWindowEnd = windowEnd
	}
	return result, nil
}

func (s *SparkSync) fetchIncrementalOnce(ctx context.Context, cursor SyncCursor, skip int, windowEnd time.Time, withExpand bool) (PageResult, error) {
	top := s.cfg.Spark.SyncIncrementalTop
	if top <= 0 {
		top = 1000
	}
	if top > 1000 {
		top = 1000
	}

	lower := cursor.LastModificationTimestamp.UTC().Format("2006-01-02T15:04:05Z")
	upper := windowEnd.UTC().Format("2006-01-02T15:04:05Z")
	filter := fmt.Sprintf("(%s) and ModificationTimestamp gt %s and ModificationTimestamp lt %s",
		activePendingStatusFilter, lower, upper)

	fetchURL := s.propertyCollectionURL()
	query := url.Values{}
	query.Set("$filter", filter)
	query.Set("$orderby", "ModificationTimestamp asc")
	query.Set("$top", fmt.Sprintf("%d", top))
	query.Set("$skip", fmt.Sprintf("%d", skip))
	if withExpand {
		s.applySyncExpand(query, false)
	}

	return s.fetchPage(ctx, fetchURL, query, cursor.DatasetSlug, false)
}

func (s *SparkSync) applySyncExpand(query url.Values, replication bool) {
	expand := strings.TrimSpace(s.cfg.MLS.SyncExpand)
	if replication {
		if repl := strings.TrimSpace(s.cfg.MLS.SyncReplicationExpand); repl != "" {
			expand = repl
		}
	}
	if expand != "" {
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

	maxRetries := s.cfg.Spark.SyncMaxHTTPRetries
	if maxRetries < 1 {
		maxRetries = 1
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return PageResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.Spark.AccessToken)
	req.Header.Set("Accept", "application/json")

	got, err := doODataGET(ctx, s.http, s.limiter, req, maxRetries, "spark")
	if err != nil {
		return PageResult{}, err
	}
	body := got.Body

	result := PageResult{
		FetchURL:    fetchURL,
		UpstreamURL: reqURL,
		ODataQuery:  odataQueryMap(query),
		HTTPStatus:  got.Status,
	}

	if got.Status == 403 {
		result.Forbidden = true
		return result, nil
	}
	if got.Status < 200 || got.Status >= 300 {
		result.HTTPError = true
		result.ODataError = sparkODataErrorMessage(body)
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
	next := extractNextURL(got.Header.Get("Link"), body)
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

func sparkODataErrorMessage(body []byte) string {
	var parsed struct {
		Error struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}
	msg := strings.TrimSpace(parsed.Error.Message)
	if msg == "" {
		msg = strings.TrimSpace(parsed.Error.Code)
	}
	if len(msg) > 500 {
		msg = msg[:500] + "…"
	}
	return msg
}

func (s *SparkSync) propertyCollectionURL() string {
	host := strings.TrimRight(s.cfg.Spark.ReplicationHost, "/")
	root := strings.Trim(s.cfg.Spark.ReplicationReso, "/")
	return host + "/" + root + "/Property"
}
