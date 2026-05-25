package mls

import "testing"

func TestResolveListPriceOrder(t *testing.T) {
	got, ok := ResolveListPrice(map[string]any{"ListPrice": 100})
	if !ok || got != 100 {
		t.Fatalf("ListPrice got %v ok=%v", got, ok)
	}
	got, ok = ResolveListPrice(map[string]any{"PreviousListPrice": 200, "OriginalListPrice": 300})
	if !ok || got != 200 {
		t.Fatalf("PreviousListPrice got %v ok=%v", got, ok)
	}
	got, ok = ResolveListPrice(map[string]any{"OriginalListPrice": 300})
	if !ok || got != 300 {
		t.Fatalf("OriginalListPrice got %v ok=%v", got, ok)
	}
	_, ok = ResolveListPrice(map[string]any{})
	if ok {
		t.Fatal("expected missing price")
	}
}

func TestResolveListPriceClampsOverflow(t *testing.T) {
	got, ok := ResolveListPrice(map[string]any{"ListPrice": maxNumeric14_2 + 1})
	if !ok || got != maxNumeric14_2 {
		t.Fatalf("got %v ok=%v", got, ok)
	}
}

func TestBathroomsTotalRejectsImplausibleInteger(t *testing.T) {
	if v := BathroomsTotal(map[string]any{"BathroomsTotalInteger": 6602}); v != nil {
		t.Fatalf("expected nil, got %v", *v)
	}
	if v := BathroomsTotal(map[string]any{"BathroomsTotalInteger": 2}); v == nil || *v != 2 {
		t.Fatalf("expected 2, got %v", v)
	}
}

func TestResolveLivingAreaSqftClamps(t *testing.T) {
	v := ResolveLivingAreaSqft(map[string]any{"LivingArea": 8505})
	if v == nil || *v != 8505 {
		t.Fatalf("got %v", v)
	}
	v = ResolveLivingAreaSqft(map[string]any{"BuildingAreaTotal": 8891})
	if v == nil || *v != 8891 {
		t.Fatalf("got %v", v)
	}
}
