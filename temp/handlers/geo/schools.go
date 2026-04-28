package geo

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"

	"github.com/xotec-solutions/xotec-datalayer/src/internal/analytics"
)

type schoolAutocompleteItem struct {
	ID             int64  `db:"id" json:"id"`
	Name           string `db:"name" json:"name"`
	NormalizedName string `db:"normalized_name" json:"normalized_name"`
	Slug           string `db:"slug" json:"slug"`
	Type           string `db:"school_type" json:"type"`
	StateID        int64  `db:"state_id" json:"state_id"`
	StateName      string `db:"state_name" json:"state_name"`
}

type schoolAutocompleteResponse struct {
	Query   string                   `json:"query"`
	Limit   int                      `json:"limit"`
	Results []schoolAutocompleteItem `json:"results"`
}

// AutocompleteSchools returns a list of matching schools for a search query.
// GET /api/v1/geo/autocomplete/schools
func (h *Handler) AutocompleteSchools(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	query, limit, ok := parseTextQuery(w, r)
	if !ok {
		return
	}
	stateID, _ := parseOptionalID(w, r, "state_id")

	// Optional filter by school type
	schoolType := strings.TrimSpace(r.URL.Query().Get("type"))
	var dbSchoolType *string
	if schoolType != "" {
		dbSchoolType = &schoolType
	}

	rawQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		h.trackAutocompleteEvent(ctx, analytics.AutocompleteTypeSchools, len(rawQuery), 0, time.Since(start).Milliseconds())
		writeJSON(w, http.StatusOK, schoolAutocompleteResponse{Query: rawQuery, Limit: limit, Results: []schoolAutocompleteItem{}})
		return
	}

	// Normalize query for better matching using database logic (similarity/ilike)
	// We use the normalized form from parseTextQuery

	// Note: We use raw query for normalized_name % $1 matching because $1 might be "Linco"
	// which doesn't normalize to "linco". Wait, parseTextQuery normalizes.
	// If user types "Lincoln", normalized is "lincoln".
	// If user types "Linc", normalized is "linc"?
	// normalize.Name("Linc") -> "linc".
	// normalize.Name("St. Pete") -> "st pete".

	sql := `
		SELECT 
			s.id, s.name, s.normalized_name, s.slug, s.school_type, s.state_id, st.name as state_name
		FROM schools s
		JOIN states st ON s.state_id = st.id
		WHERE 
			($2::bigint IS NULL OR s.state_id = $2)
			AND ($3::text IS NULL OR s.school_type::text = $3)
			AND (s.normalized_name % $1 OR s.name ILIKE $1 || '%')
		ORDER BY 
			-- Prioritize prefix matches
			CASE WHEN s.name ILIKE $1 || '%' THEN 0 ELSE 1 END,
			similarity(s.normalized_name, $1) DESC, 
			s.name ASC
		LIMIT $4
	`

	var results []schoolAutocompleteItem
	err := pgxscan.Select(ctx, h.Pool, &results, sql, query, stateID, dbSchoolType, limit)
	if err != nil {
		log.Printf("[geo] error autocompleting schools: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.trackAutocompleteEvent(ctx, analytics.AutocompleteTypeSchools, len(query), len(results), time.Since(start).Milliseconds())
	writeJSON(w, http.StatusOK, schoolAutocompleteResponse{Query: rawQuery, Limit: limit, Results: results})
}
