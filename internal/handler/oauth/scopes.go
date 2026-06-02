package oauth

import (
	"fmt"
	"sort"
	"strings"
)

var allowedMCPScopes = map[string]struct{}{
	"monitor": {},
	"comps":   {},
	"content": {},
	"api":     {},
}

const defaultScopeString = "monitor comps content api"

// ParseAndValidateScopes splits a space-separated scope string and validates each scope.
func ParseAndValidateScopes(scope string) ([]string, error) {
	if strings.TrimSpace(scope) == "" {
		return strings.Fields(defaultScopeString), nil
	}

	seen := map[string]struct{}{}
	var out []string
	for _, part := range strings.Fields(scope) {
		if part == "" || strings.HasPrefix(part, "granted_keys:") {
			continue
		}
		if _, ok := allowedMCPScopes[part]; !ok {
			return nil, fmt.Errorf("unknown scope: %s", part)
		}
		if _, dup := seen[part]; dup {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one scope is required")
	}
	sort.Strings(out)
	return out, nil
}

// ScopeString joins validated scopes for storage on tokens and auth codes.
func ScopeString(scopes []string) string {
	return strings.Join(scopes, " ")
}
