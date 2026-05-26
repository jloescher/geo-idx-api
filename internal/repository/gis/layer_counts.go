package gisrepo

import "context"

// LayerCounts holds row totals for each persistent GIS layer.
type LayerCounts struct {
	Parcels  int64
	Cities   int64
	Counties int64
	Zips     int64
}

// AllEmpty reports whether every persistent GIS table is empty.
func (c LayerCounts) AllEmpty() bool {
	return c.Parcels == 0 && c.Cities == 0 && c.Counties == 0 && c.Zips == 0
}

// NeedsBootstrap reports whether any layer still needs backfill.
func (c LayerCounts) NeedsBootstrap() bool {
	return c.Parcels == 0 || c.Cities == 0 || c.Counties == 0 || c.Zips == 0
}

// LoadLayerCounts returns current row totals for parcels and boundary tables.
func (r *Repository) LoadLayerCounts(ctx context.Context) (LayerCounts, error) {
	var counts LayerCounts
	var err error
	counts.Parcels, err = r.CountParcels(ctx, "")
	if err != nil {
		return counts, err
	}
	counts.Cities, counts.Counties, counts.Zips, err = r.CountBoundaries(ctx)
	return counts, err
}
