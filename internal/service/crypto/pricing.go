package crypto

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// PricingService refreshes CoinGecko quotes for listing enrichment.
type PricingService struct {
	cfg    config.Config
	db     *repository.DB
	logger *slog.Logger
	http   *http.Client
}

func NewPricingService(cfg config.Config, db *repository.DB, logger *slog.Logger) *PricingService {
	return &PricingService{
		cfg:    cfg,
		db:     db,
		logger: logger,
		http:   &http.Client{Timeout: cfg.Coingecko.HTTPTimeout},
	}
}

func (p *PricingService) Refresh(ctx context.Context) error {
	url := strings.TrimRight(p.cfg.Coingecko.BaseURL, "/") + "/simple/price?ids=" + strings.Join(coingeckoIDs(), ",") + "&vs_currencies=usd"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if p.cfg.Coingecko.APIKey != "" {
		req.Header.Set("x-cg-demo-api-key", p.cfg.Coingecko.APIKey)
	}
	resp, err := p.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var prices map[string]map[string]float64
	if err := json.Unmarshal(body, &prices); err != nil {
		return err
	}
	now := time.Now()
	for asset, vs := range prices {
		key, ok := assetKeyForCoingeckoID(asset)
		if !ok {
			continue
		}
		for cur, price := range vs {
			_, _ = p.db.Pool.Exec(ctx, `
				INSERT INTO crypto_price_snapshots (asset_key, vs_currency, price, captured_at, created_at, updated_at)
				VALUES ($1, $2, $3, $4, NOW(), NOW())
				ON CONFLICT (asset_key, vs_currency) DO UPDATE SET price = EXCLUDED.price, captured_at = EXCLUDED.captured_at, updated_at = NOW()
			`, key, cur, price, now)
		}
	}
	return nil
}
