package search

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// RawCursor is the decoded cursor payload.
type RawCursor struct {
	SortKey json.RawMessage `json:"sort_key"`
	PK      int64           `json:"pk"`
}

// EncodeCursor encodes the sort key and primary key into a cursor string.
func EncodeCursor(sortKey any, pk int64) (string, error) {
	payload := map[string]any{
		"sort_key": sortKey,
		"pk":       pk,
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// DecodeCursor decodes a base64url cursor.
func DecodeCursor(raw string) (RawCursor, error) {
	var cursor RawCursor
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return cursor, fmt.Errorf("cursor is empty")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return cursor, fmt.Errorf("invalid cursor encoding")
	}
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return cursor, fmt.Errorf("invalid cursor payload")
	}
	if cursor.PK == 0 {
		return cursor, fmt.Errorf("cursor pk missing")
	}
	return cursor, nil
}

// EffectiveLimit computes the per-page limit after applying overall_limit.
func EffectiveLimit(page PageParams) int {
	limit := page.Limit.IntOr(defaultPageLimit)
	if limit < 1 {
		limit = 1
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	overall := page.OverallLimit.IntOr(0)
	returned := page.Returned.IntOr(0)
	if overall <= 0 {
		return limit
	}
	remaining := overall - returned
	if remaining <= 0 {
		return 0
	}
	if remaining < limit {
		return remaining
	}
	return limit
}
