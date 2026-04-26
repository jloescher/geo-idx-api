package search

import "testing"

func TestEncodeDecodeCursor(t *testing.T) {
	cursor, err := EncodeCursor(123.45, 99)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}
	decoded, err := DecodeCursor(cursor)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	if decoded.PK != 99 {
		t.Fatalf("expected pk 99, got %d", decoded.PK)
	}
}

func TestEffectiveLimit(t *testing.T) {
	limit := EffectiveLimit(PageParams{})
	if limit != 24 {
		t.Fatalf("expected default limit 24, got %d", limit)
	}

	limit = EffectiveLimit(PageParams{Limit: IntParam{Value: intValuePtr(10)}})
	if limit != 10 {
		t.Fatalf("expected limit 10, got %d", limit)
	}

	limit = EffectiveLimit(PageParams{Limit: IntParam{Value: intValuePtr(24)}, OverallLimit: IntParam{Value: intValuePtr(50)}, Returned: IntParam{Value: intValuePtr(48)}})
	if limit != 2 {
		t.Fatalf("expected limit 2, got %d", limit)
	}

	limit = EffectiveLimit(PageParams{Limit: IntParam{Value: intValuePtr(24)}, OverallLimit: IntParam{Value: intValuePtr(10)}, Returned: IntParam{Value: intValuePtr(10)}})
	if limit != 0 {
		t.Fatalf("expected limit 0, got %d", limit)
	}
}

func intValuePtr(v int) *int {
	return &v
}
