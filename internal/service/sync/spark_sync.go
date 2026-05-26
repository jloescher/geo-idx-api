package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
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
			ReplicationComplete:  true,
			IncrementalWindowEnd: windowEnd,
		}, nil
	}

	result, err := s.fetchIncrementalOnce(ctx, cursor, skip, *windowEnd, sparkIncrementalWithExpand)
	if err != nil {
		return result, err
	}
	if result.HTTPError && result.HTTPStatus == http.StatusBadRequest {
		retry, retryErr := s.fetchIncrementalOnce(ctx, cursor, skip, *windowEnd, sparkIncrementalNoExpand)
		if retryErr != nil {
			return result, retryErr
		}
		if !retry.HTTPError {
			result = retry
		} else if retry.HTTPStatus == http.StatusBadRequest {
			bare, bareErr := s.fetchIncrementalOnce(ctx, cursor, skip, *windowEnd, sparkIncrementalBare)
			if bareErr != nil {
				return result, bareErr
			}
			if !bare.HTTPError {
				result = bare
			} else {
				result = coalesceSparkIncremental400Failures(result, retry, bare)
			}
		} else {
			result = retry
		}
	}
	if cursor.IncrementalWindowEnd == nil {
		result.IncrementalWindowEnd = windowEnd
	}
	return result, nil
}

// incremental fetch shapes: Spark sometimes returns HTTP 400 on incremental with $expand
// and/or $orderby; we retry with simpler queries.
type sparkIncrementalVariant int

const (
	sparkIncrementalWithExpand sparkIncrementalVariant = iota
	sparkIncrementalNoExpand
	sparkIncrementalBare
)

func (s *SparkSync) fetchIncrementalOnce(ctx context.Context, cursor SyncCursor, skip int, windowEnd time.Time, variant sparkIncrementalVariant) (PageResult, error) {
	top := s.cfg.Spark.SyncIncrementalTop
	if top <= 0 {
		top = 1000
	}
	if top > 1000 {
		top = 1000
	}
	if variant == sparkIncrementalBare {
		if top > 250 {
			top = 250
		}
	}

	lower := cursor.LastModificationTimestamp.UTC().Format("2006-01-02T15:04:05Z")
	upper := windowEnd.UTC().Format("2006-01-02T15:04:05Z")
	// activePendingStatusFilter is already parenthesized; avoid ((...)) which some OData stacks reject.
	filter := fmt.Sprintf("%s and ModificationTimestamp gt %s and ModificationTimestamp lt %s",
		activePendingStatusFilter, lower, upper)

	fetchURL := s.propertyCollectionURL()
	query := url.Values{}
	query.Set("$filter", filter)
	if variant != sparkIncrementalBare {
		query.Set("$orderby", "ModificationTimestamp asc")
	}
	query.Set("$top", fmt.Sprintf("%d", top))
	query.Set("$skip", fmt.Sprintf("%d", skip))
	if variant == sparkIncrementalWithExpand {
		s.applySyncExpand(query, false)
	}

	return s.fetchPage(ctx, fetchURL, query, cursor.DatasetSlug, false)
}

func coalesceSpark400Messages(parts ...string) string {
	var b strings.Builder
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(p)
	}
	return b.String()
}

func coalesceSparkIncremental400Failures(expand400, noExpand400, bare400 PageResult) PageResult {
	out := bare400
	out.HTTPError = true
	out.ODataError = coalesceSpark400Messages(
		expand400.ODataError,
		noExpand400.ODataError,
		bare400.ODataError,
	)
	if out.ODataError == "" {
		out.ODataError = "Spark returned HTTP 400 on incremental (expand, no-expand, and bare attempts)"
	}
	return out
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
		ct := strings.ToLower(got.Header.Get("Content-Type"))
		result.ODataError = sparkHTTPErrorDetail(body, ct)
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

var (
	// Keep this regexp permissive; we trim/snippet the extracted text afterwards.
	// Using bounded repeats (e.g. {1,2000}) can trip Go's RE2 parser depending on grouping.
	sparkXMLMessageRE = regexp.MustCompile(`(?i)<(?:m:)?message[^>]*>([^<]+)</(?:m:)?message>`)
)

func sparkHTTPErrorDetail(body []byte, contentType string) string {
	if msg := sparkODataJSONError(body); msg != "" {
		return msg
	}
	if strings.Contains(contentType, "xml") || looksLikeXML(body) {
		if m := sparkXMLMessageRE.FindSubmatch(body); len(m) > 1 {
			return strings.TrimSpace(string(m[1]))
		}
	}
	return sparkBodySnippet(body)
}

func looksLikeXML(body []byte) bool {
	b := strings.TrimSpace(string(body))
	return strings.HasPrefix(b, "<?xml") || strings.HasPrefix(b, "<") && strings.Contains(b, "<error")
}

func sparkODataJSONError(body []byte) string {
	var envelope struct {
		Error struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error"`
		ODataError struct {
			Message *struct {
				Lang  string `json:"lang"`
				Value string `json:"value"`
			} `json:"message"`
			Code string `json:"code"`
		} `json:"odata.error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return ""
	}
	msg := strings.TrimSpace(envelope.Error.Message)
	if msg == "" {
		msg = strings.TrimSpace(envelope.Error.Code)
	}
	if msg == "" && envelope.ODataError.Message != nil {
		msg = strings.TrimSpace(envelope.ODataError.Message.Value)
	}
	if msg == "" {
		msg = strings.TrimSpace(envelope.ODataError.Code)
	}
	if len(msg) > 500 {
		msg = msg[:500] + "…"
	}
	return msg
}

func sparkBodySnippet(body []byte) string {
	s := strings.TrimSpace(string(body))
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 400 {
		s = s[:400] + "…"
	}
	return s
}

// FetchReconcileKeysPage loads ListingKey values for mirror reconciliation (AP-filtered replication catalog).
func (s *SparkSync) FetchReconcileKeysPage(ctx context.Context, dataset string, nextURL *string) (KeyPageResult, error) {
	var fetchURL string
	query := url.Values{}
	if nextURL != nil && strings.TrimSpace(*nextURL) != "" {
		fetchURL = *nextURL
	} else {
		fetchURL = s.propertyCollectionURL()
		query.Set("$filter", SparkReplicationFilter(s.cfg))
		top := s.cfg.Spark.SyncReplicationTop
		if top <= 0 {
			top = 1000
		}
		if top > 1000 {
			top = 1000
		}
		query.Set("$top", fmt.Sprintf("%d", top))
		query.Set("$select", "ListingKey")
	}

	page, err := s.fetchPage(ctx, fetchURL, query, dataset, true)
	if err != nil {
		return KeyPageResult{}, err
	}
	return sparkKeyPageFromResult(page), nil
}

func sparkKeyPageFromResult(page PageResult) KeyPageResult {
	out := KeyPageResult{
		Keys:       dedupeListingKeys(listingKeysFromRows(page.Rows)),
		NextURL:    page.NextReplicationURL,
		HTTPError:  page.HTTPError,
		HTTPStatus: page.HTTPStatus,
		ODataError: page.ODataError,
		FetchURL:   page.FetchURL,
	}
	if page.HTTPError {
		return out
	}
	out.Complete = page.ReplicationComplete
	return out
}

func (s *SparkSync) propertyCollectionURL() string {
	host := strings.TrimRight(s.cfg.Spark.ReplicationHost, "/")
	root := strings.Trim(s.cfg.Spark.ReplicationReso, "/")
	return host + "/" + root + "/Property"
}
