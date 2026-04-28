package search

import (
	"encoding/json"
	"testing"

	searchsvc "github.com/xotec-solutions/xotec-datalayer/src/internal/search"
)

func TestParseSearchPayload_WithCurrencyTickers(t *testing.T) {
	// Test with currency_tickers at top level
	payload := []byte(`{
		"params": {"min_price": 100000},
		"page": {"limit": 24},
		"currency_tickers": ["BTC", "ETH", "USD", "EUR"]
	}`)

	req, err := searchsvc.ParseSearchPayload(payload)
	if err != nil {
		t.Fatalf("parse with currency_tickers failed: %v", err)
	}

	if len(req.CurrencyTickers) != 4 {
		t.Errorf("expected 4 currency tickers, got %d", len(req.CurrencyTickers))
	}

	expected := []string{"BTC", "ETH", "USD", "EUR"}
	for i, ticker := range req.CurrencyTickers {
		if ticker != expected[i] {
			t.Errorf("expected ticker %s at index %d, got %s", expected[i], i, ticker)
		}
	}
}

func TestParseSearchPayload_WithContextCurrencyTickers(t *testing.T) {
	// Test with currency_tickers in context
	payload := []byte(`{
		"context": {
			"params": {"min_price": 100000},
			"currency_tickers": ["BTC", "USD"]
		},
		"page": {"limit": 24}
	}`)

	req, err := searchsvc.ParseSearchPayload(payload)
	if err != nil {
		t.Fatalf("parse with context currency_tickers failed: %v", err)
	}

	if len(req.CurrencyTickers) != 2 {
		t.Errorf("expected 2 currency tickers from context, got %d", len(req.CurrencyTickers))
	}
}

func TestParseSearchPayload_NoCurrencyTickers(t *testing.T) {
	// Test without currency_tickers
	payload := []byte(`{
		"params": {"min_price": 100000},
		"page": {"limit": 24}
	}`)

	req, err := searchsvc.ParseSearchPayload(payload)
	if err != nil {
		t.Fatalf("parse without currency_tickers failed: %v", err)
	}

	if len(req.CurrencyTickers) != 0 {
		t.Errorf("expected 0 currency tickers, got %d", len(req.CurrencyTickers))
	}
}

func TestSearchResponse_WithCurrencyQuotes(t *testing.T) {
	// Test that searchResponse can include currency quotes
	resp := searchResponse{
		Status:  "ok",
		Results: nil,
		HasMore: false,
		Count:   10,
		CurrencyQuotes: map[string]float64{
			"BTC": 45000.50,
			"ETH": 3200.75,
			"EUR": 0.92,
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal searchResponse failed: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	quotes, ok := decoded["currency_quotes"].(map[string]any)
	if !ok {
		t.Fatal("currency_quotes not found in response")
	}

	if quotes["BTC"] != 45000.5 {
		t.Errorf("expected BTC price 45000.5, got %v", quotes["BTC"])
	}
}

func TestSearchResponse_WithoutCurrencyQuotes(t *testing.T) {
	// Test that searchResponse omits currency_quotes when empty
	resp := searchResponse{
		Status:  "ok",
		Results: nil,
		HasMore: false,
		Count:   10,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal searchResponse failed: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, exists := decoded["currency_quotes"]; exists {
		t.Error("currency_quotes should be omitted when empty")
	}
}
