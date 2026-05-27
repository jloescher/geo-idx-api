package comps

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/quantyralabs/idx-api/internal/repository"
	compscache "github.com/quantyralabs/idx-api/internal/repository/comps_cache"
	"github.com/quantyralabs/idx-api/internal/service/mls"
)

func (e *Engine) fetchSoldComps(
	ctx context.Context,
	domainSlug string,
	feed mls.FeedDefinition,
	subject SubjectProfile,
	scope ScopeInput,
	filters FiltersInput,
	limit int,
) ([]CompRecord, error) {
	if limit <= 0 {
		limit = 25
	}
	months := 12
	if filters.SoldMonthsBack != nil && *filters.SoldMonthsBack > 0 {
		months = *filters.SoldMonthsBack
	}
	cutoff := time.Now().AddDate(0, -months, 0).UTC()
	maxAge := time.Duration(e.cfg.Comps.ClosedCacheDays) * 24 * time.Hour
	if maxAge <= 0 {
		maxAge = 30 * 24 * time.Hour
	}

	minHits := limit
	if e.cfg.Comps.ClosedCacheMinHits > 0 {
		minHits = e.cfg.Comps.ClosedCacheMinHits
	}
	if minHits < 3 {
		minHits = 3
	}

	repo := compscache.NewRepository(e.db.Pool)
	if domainSlug != "" {
		cached, err := repo.FindClosedInScope(ctx, domainSlug, feed.Code, subject.Lat, subject.Lng,
			compscache.ScopeQuery{Type: scope.Type, RadiusMiles: scope.RadiusMiles, PostalCodes: scope.PostalCodes},
			cutoff, maxAge, limit*2)
		if err != nil {
			slog.Warn("comps closed cache read", "error", err, "domain", domainSlug)
		} else if len(cached) >= minHits {
			out := compsFromCached(cached, subject, scope)
			if len(out) >= minHits {
				if len(out) > limit {
					out = out[:limit]
				}
				slog.Debug("comps closed cache hit", "domain", domainSlug, "feed", feed.Code, "count", len(out))
				return out, nil
			}
		}
	}

	sold, rawRows, err := e.fetchSoldCompsLive(ctx, feed, subject, scope, filters, limit)
	if err != nil {
		return nil, err
	}
	if domainSlug != "" && len(rawRows) > 0 {
		if err := e.writeThroughClosedCache(ctx, repo, domainSlug, feed.Code, rawRows); err != nil {
			slog.Warn("comps closed cache write-through", "error", err, "domain", domainSlug)
		}
	}
	return sold, nil
}

func compsFromCached(rows []compscache.CachedClosedRow, subject SubjectProfile, scope ScopeInput) []CompRecord {
	var out []CompRecord
	for _, row := range rows {
		body, err := compscache.PayloadBytes(row.CompressedPayload)
		if err != nil {
			continue
		}
		c := parseProperty(json.RawMessage(body))
		if c.ListingKey == "" {
			c.ListingKey = row.ListingKey
		}
		if c.ListingKey == subject.ListingKey {
			continue
		}
		if c.ClosePrice <= 0 {
			continue
		}
		if c.Lat == 0 && c.Lng == 0 {
			c.Lat, c.Lng = row.Latitude, row.Longitude
		}
		c.DistanceMiles = haversineMiles(subject.Lat, subject.Lng, c.Lat, c.Lng)
		if scope.Type != "zip" && scope.RadiusMiles != nil && c.DistanceMiles > *scope.RadiusMiles*1.15 {
			continue
		}
		out = append(out, c)
	}
	return out
}

func (e *Engine) writeThroughClosedCache(ctx context.Context, repo *compscache.Repository, domainSlug, feedCode string, rawRows []json.RawMessage) error {
	upserts := make([]compscache.ClosedUpsertRow, 0, len(rawRows))
	for _, raw := range rawRows {
		c := parseProperty(raw)
		if c.ListingKey == "" || c.ClosePrice <= 0 {
			continue
		}
		normalized := mls.SanitizeUpstreamPropertyJSONForInternal(raw)
		row := compscache.ClosedUpsertRow{
			ListingKey:     c.ListingKey,
			StandardStatus: strings.ToLower(strings.TrimSpace(c.StandardStatus)),
			ClosePrice:     &c.ClosePrice,
			PayloadRaw:     normalized,
		}
		if row.StandardStatus == "" {
			row.StandardStatus = "closed"
		}
		if c.Lat != 0 || c.Lng != 0 {
			lat, lng := c.Lat, c.Lng
			row.Latitude, row.Longitude = &lat, &lng
		}
		if t := parseCloseDate(c.CloseDate); t != nil {
			row.CloseDate = t
		}
		upserts = append(upserts, row)
	}
	return repo.UpsertClosedBatch(ctx, domainSlug, feedCode, upserts)
}

func parseCloseDate(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			utc := t.UTC()
			return &utc
		}
	}
	return nil
}

// PurgeClosedListingsCache removes expired closed comp rows (scheduler/worker).
func PurgeClosedListingsCache(ctx context.Context, db *repository.DB, days int) (int64, error) {
	if days <= 0 {
		days = 30
	}
	return compscache.NewRepository(db.Pool).PurgeExpired(ctx, days)
}
