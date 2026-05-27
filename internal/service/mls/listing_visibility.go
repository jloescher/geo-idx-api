package mls

import (
	"encoding/json"
	"strings"
)

// VisibilitySurface controls how IDX/Internet flags apply to API output.
type VisibilitySurface int

const (
	VisibilityCompsInternal VisibilitySurface = iota
	VisibilityPublicSearch
	VisibilityHomeValuePublic
)

// ListingIDXFlags are typed mirror IDX/Internet columns used for visibility decisions.
type ListingIDXFlags struct {
	DatasetSlug                      string
	InternetEntireListingDisplayYN   *bool
	InternetAddressDisplayYN         *bool
	InternetAutomatedValuationDisplayYN *bool
	InternetConsumerCommentYN        *bool
	IDXParticipationYN               *bool
}

// IDXFlagsFromMirrorRow extracts visibility flags from a mirror row.
func IDXFlagsFromMirrorRow(row MirrorListingRow) ListingIDXFlags {
	return ListingIDXFlags{
		DatasetSlug:                        row.DatasetSlug,
		InternetEntireListingDisplayYN:     row.InternetEntireListingDisplayYN,
		InternetAddressDisplayYN:           row.InternetAddressDisplayYN,
		InternetAutomatedValuationDisplayYN: row.InternetAutomatedValuationDisplayYN,
		InternetConsumerCommentYN:          row.InternetConsumerCommentYN,
		IDXParticipationYN:                 row.IDXParticipationYN,
	}
}

// IDXFlagsFromMap reads IDX flags from a RESO Property map (upstream/JSON).
func IDXFlagsFromMap(m map[string]any, datasetSlug string) ListingIDXFlags {
	return ListingIDXFlags{
		DatasetSlug:                        datasetSlug,
		InternetEntireListingDisplayYN:     boolPtr(m["InternetEntireListingDisplayYN"]),
		InternetAddressDisplayYN:           boolPtr(m["InternetAddressDisplayYN"]),
		InternetAutomatedValuationDisplayYN: boolPtr(m["InternetAutomatedValuationDisplayYN"]),
		InternetConsumerCommentYN:          boolPtr(m["InternetConsumerCommentYN"]),
		IDXParticipationYN:                 boolPtr(m["IDXParticipationYN"]),
	}
}

// IsListingPublicCompliant reports whether a listing may appear on public IDX search surfaces.
func IsListingPublicCompliant(flags ListingIDXFlags) bool {
	if flags.InternetEntireListingDisplayYN != nil && !*flags.InternetEntireListingDisplayYN {
		return false
	}
	if strings.EqualFold(flags.DatasetSlug, "stellar") {
		if flags.IDXParticipationYN != nil && !*flags.IDXParticipationYN {
			return false
		}
	}
	return true
}

// ApplyPublicListingVisibility mutates root for public search (field omit + media filter).
// Returns false when the entire listing must be dropped from public search results.
func ApplyPublicListingVisibility(root map[string]any, flags ListingIDXFlags, surface VisibilitySurface) bool {
	switch surface {
	case VisibilityCompsInternal:
		return true
	case VisibilityHomeValuePublic:
		if !IsListingPublicCompliant(flags) {
			return false
		}
		return true
	case VisibilityPublicSearch:
		if !IsListingPublicCompliant(flags) {
			return false
		}
		omitAddressFieldsWhenNeeded(root, flags)
		omitAVMExtensionFields(root, flags)
		filterMediaForPublicSearch(root)
		return true
	default:
		return true
	}
}

func omitAddressFieldsWhenNeeded(root map[string]any, flags ListingIDXFlags) {
	if flags.InternetAddressDisplayYN == nil || *flags.InternetAddressDisplayYN {
		return
	}
	for _, k := range []string{
		"UnparsedAddress", "StreetNumber", "StreetName", "UnitNumber",
		"StreetDirPrefix", "StreetDirSuffix", "StreetSuffix",
	} {
		delete(root, k)
	}
}

func omitAVMExtensionFields(root map[string]any, flags ListingIDXFlags) {
	if flags.InternetAutomatedValuationDisplayYN == nil || *flags.InternetAutomatedValuationDisplayYN {
		return
	}
	for k := range root {
		if strings.Contains(k, "Valuation") || strings.Contains(k, "AutomatedValuation") {
			delete(root, k)
		}
	}
}

// filterMediaForPublicSearch drops media items that are not Public/IDX per RESO Permission.
func filterMediaForPublicSearch(root map[string]any) {
	raw, ok := root["Media"]
	if !ok {
		return
	}
	arr, ok := raw.([]any)
	if !ok {
		return
	}
	filtered := make([]any, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			filtered = append(filtered, item)
			continue
		}
		if mediaItemAllowedForPublicIDX(m) {
			filtered = append(filtered, m)
		}
	}
	if len(filtered) == 0 {
		delete(root, "Media")
		return
	}
	root["Media"] = filtered
}

func mediaItemAllowedForPublicIDX(m map[string]any) bool {
	perms := permissionTokens(m["Permission"])
	if len(perms) == 0 {
		return true
	}
	for _, p := range perms {
		switch strings.ToLower(strings.TrimSpace(p)) {
		case "public", "idx":
			return true
		}
	}
	return false
}

func permissionTokens(v any) []string {
	switch t := v.(type) {
	case string:
		if strings.TrimSpace(t) == "" {
			return nil
		}
		return []string{t}
	case []any:
		var out []string
		for _, item := range t {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return t
	default:
		return nil
	}
}

// ApplyPublicListingVisibilityJSON applies public search visibility to marshaled property JSON.
func ApplyPublicListingVisibilityJSON(body json.RawMessage, flags ListingIDXFlags, surface VisibilitySurface) (json.RawMessage, bool) {
	if len(body) == 0 {
		return body, true
	}
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return body, true
	}
	if !ApplyPublicListingVisibility(root, flags, surface) {
		return nil, false
	}
	omitNullTopLevel(root)
	out, err := json.Marshal(root)
	if err != nil {
		return body, true
	}
	return out, true
}

// FilterCompsForPublicHomeValue removes non-compliant comps from a public home_value response.
func FilterCompsForPublicHomeValue(comps []json.RawMessage, datasetSlug string) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(comps))
	for _, raw := range comps {
		var root map[string]any
		if err := json.Unmarshal(raw, &root); err != nil {
			continue
		}
		flags := IDXFlagsFromMap(root, datasetSlug)
		if !IsListingPublicCompliant(flags) {
			continue
		}
		out = append(out, raw)
	}
	return out
}
