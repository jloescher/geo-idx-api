package crypto

import (
	"context"
	"encoding/json"
	"time"

	"github.com/quantyralabs/idx-api/internal/repository"
)

// PricingReader loads latest crypto snapshots for optional listing enrichment.
type PricingReader struct {
	db *repository.DB
}

func NewPricingReader(db *repository.DB) *PricingReader {
	return &PricingReader{db: db}
}

// PricePoint is a single asset quote with capture time.
type PricePoint struct {
	AssetKey   string    `json:"asset_key"`
	Price      float64   `json:"price"`
	CapturedAt time.Time `json:"captured_at"`
}

// LatestPrices returns USD prices with capture timestamps.
func (p *PricingReader) LatestPrices(ctx context.Context) ([]PricePoint, error) {
	pool, err := p.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT asset_key, price, captured_at FROM crypto_price_snapshots WHERE vs_currency = 'usd'
		ORDER BY asset_key
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PricePoint
	for rows.Next() {
		var pt PricePoint
		if err := rows.Scan(&pt.AssetKey, &pt.Price, &pt.CapturedAt); err != nil {
			return nil, err
		}
		out = append(out, pt)
	}
	return out, rows.Err()
}

// LatestSnapshot returns asset prices keyed by asset_key (btc, eth, sol, xrp).
func (p *PricingReader) LatestSnapshot(ctx context.Context) (map[string]float64, error) {
	pool, err := p.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT asset_key, price FROM crypto_price_snapshots WHERE vs_currency = 'usd'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]float64)
	for rows.Next() {
		var key string
		var price float64
		if err := rows.Scan(&key, &price); err != nil {
			return nil, err
		}
		out[key] = price
	}
	return out, rows.Err()
}

// InjectIntoJSON adds pricing and pricing_converted when snapshot data exists.
func (p *PricingReader) InjectIntoJSON(ctx context.Context, body []byte) []byte {
	snap, err := p.LatestSnapshot(ctx)
	if err != nil || len(snap) == 0 {
		return body
	}
	var doc map[string]any
	if err := json.Unmarshal(body, &doc); err != nil {
		return body
	}
	doc["pricing"] = snap
	doc["pricing_converted"] = snap
	b, err := json.Marshal(doc)
	if err != nil {
		return body
	}
	return b
}
