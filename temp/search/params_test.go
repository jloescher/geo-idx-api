package search

import (
	"encoding/json"
	"testing"
)

func TestParseSearchPayload_FlatAndNested(t *testing.T) {
	flat := []byte(`{"params":{"min_price":"350000"},"page":{"limit":24}}`)
	req, err := ParseSearchPayload(flat)
	if err != nil {
		t.Fatalf("flat parse failed: %v", err)
	}
	if req.Params.MinPrice.Value == nil || *req.Params.MinPrice.Value != 350000 {
		t.Fatalf("min_price not parsed: %+v", req.Params.MinPrice.Value)
	}

	nested := []byte(`{"context":{"params":{"min_beds":2}},"page":{"limit":12}}`)
	req, err = ParseSearchPayload(nested)
	if err != nil {
		t.Fatalf("nested parse failed: %v", err)
	}
	if req.Params.MinBeds.Value == nil || *req.Params.MinBeds.Value != 2 {
		t.Fatalf("min_beds not parsed: %+v", req.Params.MinBeds.Value)
	}
}

func TestParseSearchPayload_UnknownField(t *testing.T) {
	payload := []byte(`{"params":{"unknown":123}}`)
	_, err := ParseSearchPayload(payload)
	if err == nil {
		t.Fatal("expected error")
	}
	vErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if _, exists := vErr.Fields["params.unknown"]; !exists {
		t.Fatalf("expected unknown field error, got %#v", vErr.Fields)
	}
}

func TestParseSearchPayload_PageValidation(t *testing.T) {
	payload := []byte(`{"page":{"limit":500}}`)
	_, err := ParseSearchPayload(payload)
	if err == nil {
		t.Fatal("expected validation error")
	}
	vErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if _, exists := vErr.Fields["page.limit"]; !exists {
		t.Fatalf("expected page.limit error, got %#v", vErr.Fields)
	}
}

func TestParseSearchPayload_OverallLimit(t *testing.T) {
	payload := []byte(`{"page":{"overall_limit":10,"returned":12}}`)
	_, err := ParseSearchPayload(payload)
	if err == nil {
		t.Fatal("expected validation error")
	}
	vErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if _, exists := vErr.Fields["page.returned"]; !exists {
		t.Fatalf("expected returned error, got %#v", vErr.Fields)
	}
}

func TestParseSearchPayload_SpecialListingConditions(t *testing.T) {
	payload := []byte(`{"params":{"special_listing_conditions":["Short Sale","Bank Owned"]}}`)
	req, err := ParseSearchPayload(payload)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(req.Params.SpecialListingConditions) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(req.Params.SpecialListingConditions))
	}
	if req.Params.SpecialListingConditions[0] != "Short Sale" {
		t.Fatalf("expected 'Short Sale', got %q", req.Params.SpecialListingConditions[0])
	}
	if req.Params.SpecialListingConditions[1] != "Bank Owned" {
		t.Fatalf("expected 'Bank Owned', got %q", req.Params.SpecialListingConditions[1])
	}
}

func TestParseSearchPayload_SpecialListingConditionsEmpty(t *testing.T) {
	payload := []byte(`{"params":{"special_listing_conditions":[]}}`)
	req, err := ParseSearchPayload(payload)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(req.Params.SpecialListingConditions) != 0 {
		t.Fatalf("expected 0 conditions, got %d", len(req.Params.SpecialListingConditions))
	}
}

func TestParseSearchPayload_PriceReducedWithinDays(t *testing.T) {
	payload := []byte(`{"params":{"price_reduced_within_days":7}}`)
	req, err := ParseSearchPayload(payload)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	val, ok := req.Params.PriceReducedWithinDays.Int()
	if !ok || val != 7 {
		t.Fatalf("expected 7, got %v (ok=%v)", val, ok)
	}
}

func TestParseSearchPayload_PriceReducedWithinDaysString(t *testing.T) {
	payload := []byte(`{"params":{"price_reduced_within_days":"30"}}`)
	req, err := ParseSearchPayload(payload)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	val, ok := req.Params.PriceReducedWithinDays.Int()
	if !ok || val != 30 {
		t.Fatalf("expected 30, got %v (ok=%v)", val, ok)
	}
}

func TestParseSearchPayload_PriceReducedWithinDaysNull(t *testing.T) {
	payload := []byte(`{"params":{"price_reduced_within_days":null}}`)
	req, err := ParseSearchPayload(payload)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	_, ok := req.Params.PriceReducedWithinDays.Int()
	if ok {
		t.Fatal("expected no value for null")
	}
}

func TestParseSearchPayload_IncludeRTO(t *testing.T) {
	payload := []byte(`{"include_rto":true,"params":{"min_price":100000}}`)
	req, err := ParseSearchPayload(payload)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !req.IncludeRTO {
		t.Fatal("expected IncludeRTO to be true")
	}
}

func TestParseSearchPayload_IncludeRTODefault(t *testing.T) {
	payload := []byte(`{"params":{"min_price":100000}}`)
	req, err := ParseSearchPayload(payload)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if req.IncludeRTO {
		t.Fatal("expected IncludeRTO to be false by default")
	}
}

func TestBoolParamCoercion(t *testing.T) {
	cases := map[string]bool{
		`"true"`:  true,
		`"1"`:     true,
		`"yes"`:   true,
		`"false"`: false,
		`0`:       false,
	}
	for raw, want := range cases {
		var p BoolParam
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			t.Fatalf("unmarshal %s: %v", raw, err)
		}
		got, ok := p.Bool()
		if !ok || got != want {
			t.Fatalf("unexpected bool for %s: %v", raw, got)
		}
	}
}
