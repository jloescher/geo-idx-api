package gisrepo

import "testing"

func TestShapefileAPIStatus(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		row    SourceStateRow
		expect string
	}{
		{
			name: "import done",
			row: SourceStateRow{
				SyncMode:         "shapefile",
				LastImportStatus: "done",
			},
			expect: "reachable",
		},
		{
			name: "parcels loaded",
			row: SourceStateRow{
				SyncMode:    "shapefile",
				ParcelCount: 437102,
			},
			expect: "reachable",
		},
		{
			name: "failed import no parcels",
			row: SourceStateRow{
				SyncMode:         "shapefile",
				LastImportStatus: "failed",
			},
			expect: "unreachable",
		},
		{
			name: "awaiting upload",
			row: SourceStateRow{
				SyncMode: "shapefile",
			},
			expect: "unknown",
		},
		{
			name: "stale http probe ignored when parcels exist",
			row: SourceStateRow{
				SyncMode:         "shapefile",
				ParcelCount:      100,
				LastImportStatus: "done",
			},
			expect: "reachable",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := shapefileAPIStatus(tc.row); got != tc.expect {
				t.Fatalf("shapefileAPIStatus() = %q, want %q", got, tc.expect)
			}
		})
	}
}
