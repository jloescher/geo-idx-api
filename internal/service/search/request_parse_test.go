package search

import (
	"encoding/json"
	"testing"
)

func TestSearchRequest_unmarshal_userBodyTypes(t *testing.T) {
	body := `{
  "min_price": "250000",
  "max_price": "",
  "min_beds": "2",
  "max_beds": "",
  "city": "Largo",
  "statuses": ["Active", ""],
  "low_risk_floodzone": "true",
  "min_monthly_fees": "",
  "max_monthly_fees": "500",
  "sort": "<string>",
  "page": { "limit": "24", "skip": "0" }
}`
	var req SearchRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.MinPrice == nil || *req.MinPrice != 250000 {
		t.Fatalf("min_price: %v", req.MinPrice)
	}
	if req.BedsMin == nil || *req.BedsMin != 2 {
		t.Fatalf("beds min: %v", req.BedsMin)
	}
	if req.MaxMonthlyFees == nil || *req.MaxMonthlyFees != 500 {
		t.Fatalf("max monthly fees: %v", req.MaxMonthlyFees)
	}
	if req.LowRiskFloodzone == nil || !*req.LowRiskFloodzone {
		t.Fatalf("low risk floodzone: %v", req.LowRiskFloodzone)
	}
	if len(req.Statuses) != 1 || req.Statuses[0] != "Active" {
		t.Fatalf("statuses: %v", req.Statuses)
	}
	if req.Limit != 24 || req.Skip != 0 {
		t.Fatalf("pagination: limit=%d skip=%d", req.Limit, req.Skip)
	}
}
