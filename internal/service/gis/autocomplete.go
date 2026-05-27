package gis

import (
	"context"
	"fmt"
	"strings"

	gisrepo "github.com/quantyralabs/idx-api/internal/repository/gis"
)

// CitySuggestion is a city|county autocomplete result for the GIS API.
type CitySuggestion struct {
	City       string `json:"city"`
	County     string `json:"county"`
	CountySlug string `json:"county_slug"`
	Label      string `json:"label"`
}

// CountySuggestion is a county autocomplete result for the GIS API.
type CountySuggestion struct {
	County     string `json:"county"`
	CountySlug string `json:"county_slug"`
	Label      string `json:"label"`
}

// AutocompleteService resolves GIS catalog names for autocomplete and search geography.
type AutocompleteService struct {
	repo *gisrepo.Repository
}

// NewAutocompleteService creates an autocomplete service.
func NewAutocompleteService(repo *gisrepo.Repository) *AutocompleteService {
	return &AutocompleteService{repo: repo}
}

// Cities returns city|county suggestions for q.
func (s *AutocompleteService) Cities(ctx context.Context, q string, limit int) ([]CitySuggestion, error) {
	rows, err := s.repo.AutocompleteCities(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	out := make([]CitySuggestion, 0, len(rows))
	for _, r := range rows {
		countyLabel := r.CountyName
		if countyLabel == "" {
			countyLabel = r.County
		}
		label := r.CityName
		if countyLabel != "" {
			label = fmt.Sprintf("%s | %s", r.CityName, countyLabel)
		}
		out = append(out, CitySuggestion{
			City:       r.CityName,
			County:     countyLabel,
			CountySlug: r.CountySlug,
			Label:      label,
		})
	}
	return out, nil
}

// Counties returns county suggestions for q.
func (s *AutocompleteService) Counties(ctx context.Context, q string, limit int) ([]CountySuggestion, error) {
	rows, err := s.repo.AutocompleteCounties(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	out := make([]CountySuggestion, 0, len(rows))
	for _, r := range rows {
		out = append(out, CountySuggestion{
			County:     r.CountyName,
			CountySlug: r.CountySlug,
			Label:      strings.TrimSpace(r.CountyName),
		})
	}
	return out, nil
}
