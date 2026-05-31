package mls_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/quantyralabs/idx-api/internal/service/mls"
)

func TestBuildPublicListingJSONForSearchOmitsExpandedJSONB(t *testing.T) {
	status := "Active"
	row := mls.MirrorListingRow{
		ListingKey:     "stellar:abc",
		StandardStatus: &status,
		ListPrice:      450000,
		Media:          mustJSON(t, []any{map[string]any{"MediaURL": "https://example/1.jpg"}}),
		CustomFields:   mustJSON(t, map[string]any{"STELLAR_Foo": "bar"}),
	}
	out, ok := mls.BuildPublicListingJSONForSearch(row)
	if !ok {
		t.Fatal("expected visible listing")
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if _, has := m["Media"]; has {
		t.Fatal("search JSON must not include Media")
	}
	if _, has := m["STELLAR_Foo"]; has {
		t.Fatal("search JSON must not merge custom_fields")
	}
}

func TestBuildPublicListingJSONShape(t *testing.T) {
	status := "Active"
	city := "Tampa"
	row := mls.MirrorListingRow{
		ListingKey:      "stellar:abc",
		StandardStatus:  &status,
		ListPrice:       450000,
		City:            &city,
		CustomFields: mustJSON(t, map[string]any{"STELLAR_Foo": "bar", "Room": []any{map[string]any{"RoomType": "Bed"}}}),
		Room:         mustJSON(t, []any{map[string]any{"RoomType": "Primary"}}),
	}

	out := mls.BuildPublicListingJSON(row)
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["ListingKey"] != "stellar:abc" {
		t.Fatalf("ListingKey = %#v", m["ListingKey"])
	}
	if m["STELLAR_Foo"] != "bar" {
		t.Fatalf("custom merge: STELLAR_Foo = %#v", m["STELLAR_Foo"])
	}
	if _, has := m["raw_data"]; has {
		t.Fatal("must not emit raw_data")
	}
	if _, has := m["custom_fields"]; has {
		t.Fatal("must not emit custom_fields")
	}
	if _, has := m["@odata.context"]; has {
		t.Fatal("must not emit @odata keys")
	}
	rooms, ok := m["Room"].([]any)
	if !ok || len(rooms) != 1 {
		t.Fatalf("Room from column = %#v", m["Room"])
	}
}

func TestBuildCustomFieldsStripsNavigationAliases(t *testing.T) {
	row := map[string]any{
		"ListingKey":     "k",
		"StandardStatus":   "Active",
		"ListPrice":        1,
		"STELLAR_Foo":      "bar",
		"Rooms":            []any{map[string]any{"RoomType": "Bedroom"}},
		"OpenHouses":       []any{},
		"UnitTypes":        []any{},
	}
	mls.NormalizeBridgeExpandKeys(row)
	custom := mls.BuildCustomFields(row, mls.MirrorProviderBridge, mls.ParseExpandKeys("Media,OpenHouses,Rooms,UnitTypes"), "STELLAR")
	var m map[string]any
	if err := json.Unmarshal(custom, &m); err != nil {
		t.Fatal(err)
	}
	if m["STELLAR_Foo"] != "bar" {
		t.Fatalf("custom = %#v", m)
	}
	for _, k := range []string{"Room", "Rooms", "Unit", "UnitTypes", "Units", "OpenHouse", "OpenHouses", "Media"} {
		if _, has := m[k]; has {
			t.Fatalf("navigation key %q must not be in custom_fields", k)
		}
	}
	p := mls.ExtractExpandedPayloads(row, mls.MirrorProviderBridge, mls.ParseExpandKeys("Media,OpenHouses,Rooms,UnitTypes"))
	if !p.HasRoom {
		t.Fatal("expected Room JSONB from Rooms alias")
	}
}

func TestBuildPublicListingJSONOmitsNullCustomFields(t *testing.T) {
	row := mls.MirrorListingRow{
		ListingKey: "k",
		ListPrice:  1,
		CustomFields: mustJSON(t, map[string]any{
			"STELLAR_Keep": "yes",
			"STELLAR_Null": nil,
		}),
	}
	out := mls.BuildPublicListingJSON(row)
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["STELLAR_Keep"] != "yes" {
		t.Fatalf("STELLAR_Keep = %#v", m["STELLAR_Keep"])
	}
	if _, has := m["STELLAR_Null"]; has {
		t.Fatal("null custom field must not appear at top level")
	}
}

func TestBuildPublicListingJSONDedupesProviderAliases(t *testing.T) {
	flood := "X"
	row := mls.MirrorListingRow{
		ListingKey:    "stellar:abc",
		ListPrice:     100,
		FloodZoneCode: &flood,
		CustomFields: mustJSON(t, map[string]any{
			"STELLAR_FloodZoneCode": "duplicate",
			"STELLAR_Other":         "keep",
		}),
	}
	out := mls.BuildPublicListingJSON(row)
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["FloodZoneCode"] != "X" {
		t.Fatalf("FloodZoneCode = %#v", m["FloodZoneCode"])
	}
	if _, has := m["STELLAR_FloodZoneCode"]; has {
		t.Fatal("STELLAR_FloodZoneCode should be omitted when FloodZoneCode is on root")
	}
	if m["STELLAR_Other"] != "keep" {
		t.Fatalf("STELLAR_Other = %#v", m["STELLAR_Other"])
	}
}

func TestBuildPublicListingJSONDedupesListPriceFromCustom(t *testing.T) {
	row := mls.MirrorListingRow{
		ListingKey: "k",
		ListPrice:  100,
		CustomFields: mustJSON(t, map[string]any{
			"ListPrice": float64(999),
			"STELLAR_X": "y",
		}),
	}
	out := mls.BuildPublicListingJSON(row)
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["ListPrice"] != float64(100) {
		t.Fatalf("typed ListPrice wins: %#v", m["ListPrice"])
	}
}

func TestBuildCustomFieldsStripsResolvedProviderExtensions(t *testing.T) {
	row := map[string]any{
		"ListingKey":            "k",
		"StandardStatus":        "Active",
		"ListPrice":             1,
		"STELLAR_FloodZoneCode": "X",
		"STELLAR_TotalMonthlyFees": 500.0,
		"STELLAR_OnlyInCustom":  "keep",
	}
	custom := mls.BuildCustomFields(row, mls.MirrorProviderBridge, nil, "STELLAR")
	var m map[string]any
	if err := json.Unmarshal(custom, &m); err != nil {
		t.Fatal(err)
	}
	if _, has := m["STELLAR_FloodZoneCode"]; has {
		t.Fatal("resolved flood extension should not persist in custom_fields")
	}
	if _, has := m["STELLAR_TotalMonthlyFees"]; has {
		t.Fatal("resolved fees extension should not persist in custom_fields")
	}
	if m["STELLAR_OnlyInCustom"] != "keep" {
		t.Fatalf("custom = %#v", m)
	}
}

func TestSanitizeUpstreamPropertyJSONForInternalPreservesAddress(t *testing.T) {
	raw := []byte(`{
		"@odata.context":"x",
		"ListingKey":"k",
		"InternetAddressDisplayYN":false,
		"UnparsedAddress":"123 Main St",
		"Media":[{"Permission":["Private"]}]
	}`)
	out := mls.SanitizeUpstreamPropertyJSONForInternal(raw)
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["UnparsedAddress"] != "123 Main St" {
		t.Fatalf("address stripped: %#v", m)
	}
	media, ok := m["Media"].([]any)
	if !ok || len(media) != 1 {
		t.Fatalf("media should remain for comps: %#v", m["Media"])
	}
}

func TestSanitizeUpstreamPropertyJSON(t *testing.T) {
	raw := []byte(`{"@odata.context":"x","ListingKey":"k","Rooms":[],"raw_data":{},"custom_fields":{}}`)
	out := mls.SanitizeUpstreamPropertyJSON(raw)
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if _, has := m["@odata.context"]; has {
		t.Fatal("strip odata")
	}
	if _, has := m["raw_data"]; has {
		t.Fatal("strip raw_data")
	}
	room, ok := m["Room"].([]any)
	if !ok {
		t.Fatalf("Rooms should normalize to Room, got %#v", m["Room"])
	}
	if len(room) != 0 {
		t.Fatalf("Room = %#v", room)
	}
	if _, has := m["Rooms"]; has {
		t.Fatal("Rooms alias should be removed after normalize")
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// newTestMirrorRow returns a realistic MirrorListingRow with non-null typed columns.
// Apply functional options to override specific fields for targeted test cases.
func newTestMirrorRow(overrides ...func(*mls.MirrorListingRow)) mls.MirrorListingRow {
	status := "Active"
	city := "Clearwater"
	county := "Pinellas"
	postal := "33755"
	state := "FL"
	propType := "Residential"
	propSub := "Single Family Residence"
	unparsed := "123 Test St, Clearwater FL 33755"
	remarks := "Great home with a pool."
	listID := "SLR123456"
	agentID := "AGT001"
	officeID := "OFF001"
	floatVal := func(f float64) *float64 { return &f }
	intVal := func(n int16) *int16 { return &n }
	boolVal := func(b bool) *bool { return &b }
	lat := 27.965
	lng := -82.801

	r := mls.MirrorListingRow{
		DatasetSlug:                    "stellar",
		ListingKey:                     "stellar:TEST001",
		MlsListingID:                   &listID,
		StandardStatus:                 &status,
		ListPrice:                      425000,
		BedroomsTotal:                  intVal(3),
		BathroomsTotalDecimal:          floatVal(2.5),
		LivingArea:                     floatVal(1850),
		LotSizeAcres:                   floatVal(0.18),
		City:                           &city,
		CountyOrParish:                 &county,
		PostalCode:                     &postal,
		StateOrProvince:                &state,
		PropertyType:                   &propType,
		PropertySubType:                &propSub,
		UnparsedAddress:                &unparsed,
		PublicRemarks:                  &remarks,
		ListAgentMlsID:                 &agentID,
		ListOfficeMlsID:                &officeID,
		Latitude:                       &lat,
		Longitude:                      &lng,
		PoolPrivateYN:                  boolVal(true),
		WaterfrontYN:                   boolVal(false),
		GarageYN:                       boolVal(true),
		AssociationYN:                  boolVal(true),
		EstimatedTotalMonthlyFees:      floatVal(350),
		LowRiskFloodZoneYN:             true,
		InternetEntireListingDisplayYN: boolVal(true),
		IDXParticipationYN:             boolVal(true),
	}
	for _, fn := range overrides {
		fn(&r)
	}
	return r
}

// TestBuildPublicListingJSONForSearch_Smoke runs table-driven smoke tests over
// BuildPublicListingJSONForSearch covering key output correctness cases.
func TestBuildPublicListingJSONForSearch_Smoke(t *testing.T) {
	t.Run("all typed columns present", func(t *testing.T) {
		row := newTestMirrorRow()
		out, ok := mls.BuildPublicListingJSONForSearch(row)
		if !ok {
			t.Fatal("expected visible listing")
		}
		var m map[string]any
		if err := json.Unmarshal(out, &m); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{
			"ListingKey", "StandardStatus", "ListPrice",
			"BedroomsTotal", "BathroomsTotalDecimal", "LivingArea",
			"City", "PostalCode", "StateOrProvince",
			"PropertyType", "PropertySubType",
			"Latitude", "Longitude",
			"PoolPrivateYN", "GarageYN",
		} {
			if _, has := m[key]; !has {
				t.Errorf("expected key %q in search JSON", key)
			}
		}
	})

	t.Run("expanded JSONB omitted from search variant", func(t *testing.T) {
		row := newTestMirrorRow(func(r *mls.MirrorListingRow) {
			r.Media = mustJSON(t, []any{map[string]any{"MediaURL": "https://example.com/1.jpg"}})
			r.Room = mustJSON(t, []any{map[string]any{"RoomType": "Primary Bedroom"}})
			r.Unit = mustJSON(t, []any{map[string]any{"UnitNumber": "1A"}})
			r.OpenHouse = mustJSON(t, []any{map[string]any{"OpenHouseDate": "2026-06-01"}})
		})
		out, ok := mls.BuildPublicListingJSONForSearch(row)
		if !ok {
			t.Fatal("expected visible listing")
		}
		var m map[string]any
		if err := json.Unmarshal(out, &m); err != nil {
			t.Fatal(err)
		}
		for _, absent := range []string{"Media", "Room", "Unit", "OpenHouse"} {
			if _, has := m[absent]; has {
				t.Errorf("search JSON must not include %q (search variant skips expanded JSONB)", absent)
			}
		}
	})

	t.Run("custom_fields not merged in search variant", func(t *testing.T) {
		row := newTestMirrorRow(func(r *mls.MirrorListingRow) {
			r.CustomFields = mustJSON(t, map[string]any{"STELLAR_Foo": "bar"})
		})
		out, ok := mls.BuildPublicListingJSONForSearch(row)
		if !ok {
			t.Fatal("expected visible listing")
		}
		var m map[string]any
		if err := json.Unmarshal(out, &m); err != nil {
			t.Fatal(err)
		}
		if _, has := m["STELLAR_Foo"]; has {
			t.Error("search JSON must not flat-merge custom_fields")
		}
		if _, has := m["custom_fields"]; has {
			t.Error("search JSON must not expose custom_fields envelope")
		}
	})

	t.Run("InternetEntireListingDisplayYN false suppresses listing", func(t *testing.T) {
		hidden := false
		row := newTestMirrorRow(func(r *mls.MirrorListingRow) {
			r.InternetEntireListingDisplayYN = &hidden
		})
		_, ok := mls.BuildPublicListingJSONForSearch(row)
		if ok {
			t.Error("listing with InternetEntireListingDisplayYN=false must be suppressed")
		}
	})

	t.Run("IDXParticipationYN false suppresses stellar listing", func(t *testing.T) {
		nope := false
		row := newTestMirrorRow(func(r *mls.MirrorListingRow) {
			r.DatasetSlug = "stellar"
			r.IDXParticipationYN = &nope
		})
		_, ok := mls.BuildPublicListingJSONForSearch(row)
		if ok {
			t.Error("stellar listing with IDXParticipationYN=false must be suppressed")
		}
	})

	t.Run("flood_zone block present and well-formed", func(t *testing.T) {
		// FloodZoneResponse JSON tags: mls_code, fema_code, effective_code, sfha, low_risk.
		floodCode := "AE"
		sfha := "T"
		row := newTestMirrorRow(func(r *mls.MirrorListingRow) {
			r.FloodZoneCode = &floodCode
			r.FEMAFloodZoneCode = &floodCode
			r.FloodZoneSFHA_TF = &sfha
			r.LowRiskFloodZoneYN = false
		})
		out, ok := mls.BuildPublicListingJSONForSearch(row)
		if !ok {
			t.Fatal("expected visible listing")
		}
		var m map[string]any
		if err := json.Unmarshal(out, &m); err != nil {
			t.Fatal(err)
		}
		fz, hasFZ := m["flood_zone"].(map[string]any)
		if !hasFZ {
			t.Fatalf("expected flood_zone object, got %T", m["flood_zone"])
		}
		// effective_code is mls_code when no FEMAAttemptedAt enrichment timestamp.
		if fz["mls_code"] != "AE" {
			t.Errorf("flood_zone.mls_code = %v, want AE", fz["mls_code"])
		}
		if fz["sfha"] != "T" {
			t.Errorf("flood_zone.sfha = %v, want T", fz["sfha"])
		}
		if fz["low_risk"] != false {
			t.Errorf("flood_zone.low_risk = %v, want false", fz["low_risk"])
		}
		// status and source must always be present.
		if fz["status"] == nil || fz["status"] == "" {
			t.Errorf("flood_zone.status missing or empty: %v", fz["status"])
		}
	})

	t.Run("EstimatedTotalMonthlyFees null absent from output", func(t *testing.T) {
		row := newTestMirrorRow(func(r *mls.MirrorListingRow) {
			r.EstimatedTotalMonthlyFees = nil
		})
		out, ok := mls.BuildPublicListingJSONForSearch(row)
		if !ok {
			t.Fatal("expected visible listing")
		}
		var m map[string]any
		if err := json.Unmarshal(out, &m); err != nil {
			t.Fatal(err)
		}
		if _, has := m["EstimatedTotalMonthlyFees"]; has {
			t.Error("null EstimatedTotalMonthlyFees must not appear in search JSON")
		}
	})

	t.Run("internal and OData metadata keys absent", func(t *testing.T) {
		row := newTestMirrorRow()
		out, ok := mls.BuildPublicListingJSONForSearch(row)
		if !ok {
			t.Fatal("expected visible listing")
		}
		var m map[string]any
		if err := json.Unmarshal(out, &m); err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"raw_data", "custom_fields", "@odata.context", "@odata.etag"} {
			if _, has := m[forbidden]; has {
				t.Errorf("forbidden key %q must not appear in search JSON output", forbidden)
			}
		}
	})
}

func parseMirrorSelectColumns(sql string) []string {
	sql = strings.ReplaceAll(sql, "\n", " ")
	parts := strings.Split(sql, ",")
	var cols []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			cols = append(cols, p)
		}
	}
	return cols
}

// TestMirrorListingSearchColumns_alignsWithScanOrder guards POST /search: ScanMirrorListingSearchRow
// must bind the same columns as MirrorListingSearchColumns (misalignment caused pgx 502s).
func TestMirrorListingSearchColumns_alignsWithScanOrder(t *testing.T) {
	cols := parseMirrorSelectColumns(mls.MirrorListingSearchColumns)
	if len(cols) == 0 {
		t.Fatal("expected columns")
	}
	floodIdx := -1
	for i, c := range cols {
		if c == "flood_zone_code" {
			floodIdx = i
			break
		}
	}
	if floodIdx < 0 || floodIdx+1 >= len(cols) {
		t.Fatalf("flood_zone_code not found in columns: %v", cols)
	}
	if cols[floodIdx+1] != "estimated_total_monthly_fees" {
		t.Fatalf("expected estimated_total_monthly_fees after flood_zone_code, got %q", cols[floodIdx+1])
	}
}

func TestScanMirrorListingSearchRow_readsEstimatedFees(t *testing.T) {
	wantFees := 500.0
	cols := parseMirrorSelectColumns(mls.MirrorListingSearchColumns)
	row, err := mls.ScanMirrorListingSearchRow(func(dest ...any) error {
		if len(dest) != len(cols) {
			t.Fatalf("scan dest count %d != column count %d", len(dest), len(cols))
		}
		for i, name := range cols {
			switch name {
			case "dataset_slug":
				*(dest[i].(*string)) = "stellar"
			case "listing_key":
				*(dest[i].(*string)) = "k"
			case "list_price":
				*(dest[i].(*float64)) = 1
			case "estimated_total_monthly_fees":
				*(dest[i].(**float64)) = &wantFees
			case "special_listing_conditions":
				*(dest[i].(*[]byte)) = []byte("[]")
			default:
				// leave nil pointers / zero values
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if row.EstimatedTotalMonthlyFees == nil || *row.EstimatedTotalMonthlyFees != wantFees {
		t.Fatalf("EstimatedTotalMonthlyFees = %v, want %v", row.EstimatedTotalMonthlyFees, wantFees)
	}
}
