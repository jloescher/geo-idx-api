package mls

import (
	"strings"
	"time"

	femarepo "github.com/quantyralabs/idx-api/internal/repository/fema"
)

// Flood zone client status and reason codes.
const (
	FloodZoneStatusEnriched        = "enriched"
	FloodZoneStatusMLSFallback     = "mls_fallback"
	FloodZoneStatusNoData          = "no_data"
	FloodZoneStatusOutOfCoverage   = "out_of_coverage"
	FloodZoneStatusPending         = "pending"
	FloodZoneStatusError           = "error"
	FloodZoneStatusCoordsRecovery  = "coords_recovery"

	FloodZoneReasonNFHLSuccess            = "nfhl_success"
	FloodZoneReasonNFHLNoPolygon          = "nfhl_no_polygon_at_point"
	FloodZoneReasonOutsideCoverage        = "outside_nfhl_coverage"
	FloodZoneReasonSuspiciousCoordinates  = "suspicious_coordinates"
	FloodZoneReasonUpstreamError          = "upstream_error"
	FloodZoneReasonPendingEnrichment      = "pending_enrichment"
)

// FloodZoneResponse is the nested flood_zone object on public listing/search JSON.
type FloodZoneResponse struct {
	MLSCode       *string    `json:"mls_code,omitempty"`
	FEMACode      *string    `json:"fema_code,omitempty"`
	EffectiveCode *string    `json:"effective_code,omitempty"`
	SFHA          *string    `json:"sfha,omitempty"`
	LowRisk       bool       `json:"low_risk"`
	Source        string     `json:"source"`
	Status        string     `json:"status"`
	Reason        string     `json:"reason"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

// BuildFloodZoneResponse assembles client flood metadata from mirror row columns.
func BuildFloodZoneResponse(row MirrorListingRow) FloodZoneResponse {
	effective := effectiveFloodZoneCode(row.FloodZoneCode, row.FEMAFloodZoneCode, row.FloodZoneUpdatedAt)
	source, status, reason := floodZoneClientMeta(row)

	out := FloodZoneResponse{
		MLSCode:       row.FloodZoneCode,
		FEMACode:      row.FEMAFloodZoneCode,
		EffectiveCode: effective,
		SFHA:          row.FloodZoneSFHA_TF,
		LowRisk:       row.LowRiskFloodZoneYN,
		Source:        source,
		Status:        status,
		Reason:        reason,
		UpdatedAt:     row.FloodZoneUpdatedAt,
	}
	return out
}

func floodZoneClientMeta(row MirrorListingRow) (source, status, reason string) {
	reasonStr := ""
	if row.FEMAFailureReason != nil {
		reasonStr = strings.TrimSpace(*row.FEMAFailureReason)
	}

	if reasonStr == femarepo.FailureReasonInsufficientCoords {
		return "none", FloodZoneStatusCoordsRecovery, FloodZoneReasonSuspiciousCoordinates
	}
	if reasonStr == femarepo.FailureReasonRequestError {
		return "none", FloodZoneStatusError, FloodZoneReasonUpstreamError
	}
	if reasonStr == femarepo.FailureReasonOutOfCoverage {
		if hasFloodCode(row.FloodZoneCode) {
			return "mls", FloodZoneStatusOutOfCoverage, FloodZoneReasonOutsideCoverage
		}
		return "none", FloodZoneStatusOutOfCoverage, FloodZoneReasonOutsideCoverage
	}

	hasFEMACode := row.FEMAFloodZoneCode != nil && strings.TrimSpace(*row.FEMAFloodZoneCode) != ""
	if hasFEMACode && row.FloodZoneUpdatedAt != nil {
		return "fema", FloodZoneStatusEnriched, FloodZoneReasonNFHLSuccess
	}

	if row.FloodZoneUpdatedAt != nil {
		if hasFloodCode(row.FloodZoneCode) {
			return "mls", FloodZoneStatusMLSFallback, FloodZoneReasonNFHLNoPolygon
		}
		return "none", FloodZoneStatusNoData, FloodZoneReasonNFHLNoPolygon
	}

	if hasFloodCode(row.FloodZoneCode) {
		return "mls", FloodZoneStatusPending, FloodZoneReasonPendingEnrichment
	}
	return "none", FloodZoneStatusPending, FloodZoneReasonPendingEnrichment
}

func hasFloodCode(code *string) bool {
	return code != nil && strings.TrimSpace(*code) != ""
}

func effectiveFloodZoneCode(mlsCode, femaCode *string, femaAt *time.Time) *string {
	if femaAt != nil && femaCode != nil {
		if s := strings.TrimSpace(*femaCode); s != "" {
			return femaCode
		}
	}
	return mlsCode
}
