package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// PageResult is one OData page (Bridge or Spark replication/incremental).
type PageResult struct {
	Rows                 []json.RawMessage
	NextReplicationURL   *string
	ReplicationComplete  bool
	IncrementalHasMore   bool
	NextIncrementalSkip  int
	MaxBridgeTs          *time.Time
	IncrementalWindowEnd *time.Time
	Forbidden            bool
	HTTPError            bool
	HTTPStatus           int
	FetchURL             string
	UpstreamURL          string
	ODataQuery           map[string]string
}

// BridgeSync fetches Bridge replication/incremental OData pages.
type BridgeSync struct {
	cfg     config.Config
	http    *http.Client
	cursors *CursorStore
}

func NewBridgeSync(cfg config.Config, db *repository.DB) *BridgeSync {
	return &BridgeSync{
		cfg: cfg,
		http: &http.Client{
			Timeout: cfg.Bridge.Timeout,
		},
		cursors: NewCursorStore(db),
	}
}

func (s *BridgeSync) FetchReplicationPage(ctx context.Context, dataset string, cursor SyncCursor) (PageResult, error) {
	top := s.cfg.Bridge.SyncReplicationTop
	if top <= 0 {
		top = 2000
	}

	var fetchURL string
	query := url.Values{}

	if cursor.ReplicationNextURL != nil && *cursor.ReplicationNextURL != "" {
		fetchURL = *cursor.ReplicationNextURL
	} else {
		fetchURL = s.propertyReplicationURL(dataset)
		query.Set("$filter", "(StandardStatus eq 'Active' or StandardStatus eq 'Pending')")
		query.Set("$top", fmt.Sprintf("%d", top))
		query.Set("$select", s.replicationSelectList(dataset))
	}

	return s.fetchPage(ctx, fetchURL, query, true)
}

func (s *BridgeSync) FetchIncrementalPage(ctx context.Context, dataset string, cursor SyncCursor, skip int) (PageResult, error) {
	if cursor.LastBridgeModificationTimestamp == nil {
		return PageResult{ReplicationComplete: true}, nil
	}

	top := s.cfg.Bridge.SyncIncrementalTop
	if top <= 0 {
		top = 200
	}

	ts := cursor.LastBridgeModificationTimestamp.UTC().Format("2006-01-02T15:04:05Z")
	filterLiteral := "ModificationTimestamp gt datetime'" + ts + "'"
	fetchURL := s.propertyCollectionURL(dataset)
	query := url.Values{}
	query.Set("$filter", filterLiteral)
	query.Set("$orderby", "ModificationTimestamp asc")
	query.Set("$top", fmt.Sprintf("%d", top))
	query.Set("$skip", fmt.Sprintf("%d", skip))
	query.Set("$select", s.syncSelectList(dataset))
	if !s.cfg.Bridge.SyncIncludeMedia {
		query.Set("$unselect", "Media")
	}

	result, err := s.fetchPage(ctx, fetchURL, query, false)
	if err != nil {
		return result, err
	}
	if result.HTTPError && (result.HTTPStatus == 400 || result.HTTPStatus == 501) {
		filterLiteral = "BridgeModificationTimestamp gt datetime'" + ts + "'"
		query.Set("$filter", filterLiteral)
		query.Set("$orderby", "BridgeModificationTimestamp asc")
		return s.fetchPage(ctx, fetchURL, query, false)
	}
	return result, nil
}

func (s *BridgeSync) fetchPage(ctx context.Context, fetchURL string, query url.Values, replication bool) (PageResult, error) {
	if s.cfg.Bridge.APIKey == "" {
		return PageResult{}, fmt.Errorf("BRIDGE_API_KEY is not configured")
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
	req.Header.Set("Authorization", "Bearer "+s.cfg.Bridge.APIKey)
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
		return PageResult{}, fmt.Errorf("decode bridge odata: %w", err)
	}
	rows := parsed.Value
	if rows == nil {
		rows = []json.RawMessage{}
	}

	result.Rows = rows
	result.MaxBridgeTs = maxBridgeTimestampFromRows(rows)
	next := extractNextURL(resp.Header.Get("Link"), body)
	result.NextReplicationURL = next

	if replication {
		result.ReplicationComplete = next == nil || *next == ""
		return result, nil
	}

	top := s.cfg.Bridge.SyncIncrementalTop
	if top <= 0 {
		top = 200
	}
	if len(rows) == 0 {
		result.ReplicationComplete = true
		return result, nil
	}
	result.IncrementalHasMore = len(rows) >= top
	return result, nil
}

func (s *BridgeSync) oDataPropertyBase(dataset string) string {
	host := strings.TrimRight(s.cfg.Bridge.Host, "/")
	prefix := strings.Trim(s.cfg.Bridge.PathPrefix, "/")
	resoRoot := strings.Trim(s.cfg.Bridge.ResoRoot, "/")

	var basePath string
	switch {
	case prefix != "":
		basePath = prefix + "/OData/" + dataset
	case resoRoot != "":
		basePath = resoRoot + "/OData/" + dataset
	default:
		basePath = "OData/" + dataset
	}
	return host + "/" + basePath
}

func (s *BridgeSync) propertyCollectionURL(dataset string) string {
	return s.oDataPropertyBase(dataset) + "/Property"
}

func (s *BridgeSync) propertyReplicationURL(dataset string) string {
	return s.oDataPropertyBase(dataset) + "/Property/replication"
}

func (s *BridgeSync) replicationSelectList(dataset string) string {
	fields := strings.Split(s.syncSelectList(dataset), ",")
	hasMedia := false
	for _, f := range fields {
		if strings.TrimSpace(f) == "Media" {
			hasMedia = true
			break
		}
	}
	if !hasMedia {
		fields = append(fields, "Media")
	}
	return strings.Join(fields, ",")
}

func (s *BridgeSync) syncSelectList(dataset string) string {
	upper := strings.ToUpper(dataset)
	fields := []string{
		"ListingKey", "ListingId", "BridgeModificationTimestamp", "ModificationTimestamp",
		"StandardStatus", "ListPrice", "ClosePrice", "PreviousListPrice", "PriceChangeTimestamp",
		"BedroomsTotal", "BathroomsTotalDecimal", "LivingArea", "LotSizeAcres", "YearBuilt",
		"City", "CountyOrParish", "PostalCode", "StateOrProvince", "PropertyType", "PropertySubType",
		"OnMarketDate", "CloseDate", "Latitude", "Longitude", "Coordinates",
		"WaterfrontYN", "PoolPrivateYN", "NewConstructionYN", "GarageYN", "AssociationYN",
		"SpaYN", "FireplaceYN", "SeniorCommunityYN", "SpecialListingConditions", "SubdivisionName",
		"ElementarySchool", "MiddleOrJuniorSchool", "HighSchool", "StreetNumber", "StreetName",
		"ListAgentMlsId", "ListOfficeMlsId", "Media",
		upper + "_FloodZoneCode", upper + "_TotalMonthlyFees",
	}
	if !s.cfg.Bridge.SyncIncludeMedia {
		filtered := fields[:0]
		for _, f := range fields {
			if f != "Media" {
				filtered = append(filtered, f)
			}
		}
		fields = filtered
	}
	out := make([]string, 0, len(fields))
	seen := map[string]struct{}{}
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" || f == "DockYN" {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	return strings.Join(out, ",")
}

var linkNextRE = regexp.MustCompile(`<([^>]+)>;\s*rel\s*=\s*["']?next["']?`)

func extractNextURL(linkHeader string, body []byte) *string {
	if linkHeader != "" {
		for _, segment := range strings.Split(linkHeader, ",") {
			if m := linkNextRE.FindStringSubmatch(strings.TrimSpace(segment)); len(m) == 2 {
				u := m[1]
				return &u
			}
		}
	}
	var parsed struct {
		Next string `json:"@odata.nextLink"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Next != "" {
		return &parsed.Next
	}
	return nil
}

func maxBridgeTimestampFromRows(rows []json.RawMessage) *time.Time {
	var max *time.Time
	for _, raw := range rows {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		for _, key := range []string{"ModificationTimestamp", "BridgeModificationTimestamp"} {
			ts := parseBridgeTime(m[key])
			if ts == nil {
				continue
			}
			if max == nil || ts.After(*max) {
				t := *ts
				max = &t
			}
		}
	}
	return max
}

func parseBridgeTime(v any) *time.Time {
	s, ok := v.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return nil
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.999Z",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}
