//go:build smoke

package smoke

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Case defines one smoke test endpoint check.
type Case struct {
	Name                   string            `json:"name"`
	Skip                   string            `json:"skip,omitempty"`
	Method                 string            `json:"method"`
	Path                   string            `json:"path"`
	Query                  map[string]string `json:"query,omitempty"`
	Headers                map[string]string `json:"headers,omitempty"`
	Body                   json.RawMessage   `json:"body,omitempty"`
	Auth                   *bool             `json:"auth,omitempty"`
	ExpectStatus           []int             `json:"expect_status"`
	ExpectStatusUnauthed   []int             `json:"expect_status_unauthed,omitempty"`
	ExpectJSON             map[string]string `json:"expect_json,omitempty"`
	ExpectJSONEquals       map[string]any    `json:"expect_json_equals,omitempty"`
	ExpectJSONWhenStatus   []int             `json:"expect_json_when_status,omitempty"`
	ExpectContentTypePrefix string           `json:"expect_content_type_prefix,omitempty"`
	ExpectMinBodyBytes     int               `json:"expect_min_body_bytes,omitempty"`
	ExpectJSONArray        bool              `json:"expect_json_array,omitempty"`
	ExpectJSONObject       bool              `json:"expect_json_object,omitempty"`
	ExportAs               string            `json:"export_as,omitempty"`
	NextJSHint             string            `json:"nextjs_hint,omitempty"`
}

type caseFile struct {
	Cases []Case `json:"cases"`
}

func LoadCases(dir string) ([]Case, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	var all []Case
	for _, name := range names {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		var file caseFile
		if err := json.Unmarshal(b, &file); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		all = append(all, file.Cases...)
	}
	return all, nil
}

type templateVars struct {
	ListingKey    string
	MLSListingID  string
	PhotoID       string
	Dataset       string
	DomainSlug    string
	BBox          string
	AdminEmail    string
	AdminPassword string
}

func (v templateVars) apply(s string) string {
	r := strings.NewReplacer(
		"{{listing_key}}", v.ListingKey,
		"{{mls_listing_id}}", v.MLSListingID,
		"{{photo_id}}", v.PhotoID,
		"{{dataset}}", v.Dataset,
		"{{domain_slug}}", v.DomainSlug,
		"{{bbox}}", v.BBox,
		"{{admin_email}}", v.AdminEmail,
		"{{admin_password}}", v.AdminPassword,
	)
	return r.Replace(s)
}

func (v templateVars) applyMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return m
	}
	out := make(map[string]string, len(m))
	for k, val := range m {
		out[k] = v.apply(val)
	}
	return out
}

func (c Case) resolved(vars templateVars) Case {
	out := c
	out.Path = vars.apply(c.Path)
	out.Query = vars.applyMap(c.Query)
	if len(c.Body) > 0 {
		out.Body = json.RawMessage(vars.apply(string(c.Body)))
	}
	if c.Headers != nil {
		out.Headers = make(map[string]string, len(c.Headers))
		for k, val := range c.Headers {
			out.Headers[k] = vars.apply(val)
		}
	}
	return out
}

func (c Case) wantsAuth(cfg Config) bool {
	if c.Auth != nil {
		return *c.Auth
	}
	return strings.HasPrefix(c.Path, "/api/v1/") ||
		strings.HasPrefix(c.Path, "/images/")
}

func (c Case) allowedStatuses(cfg Config) []int {
	if c.wantsAuth(cfg) && !cfg.HasAuth() && len(c.ExpectStatusUnauthed) > 0 {
		return c.ExpectStatusUnauthed
	}
	return c.ExpectStatus
}

func (c Case) shouldValidateJSON(status int) bool {
	if len(c.ExpectJSONWhenStatus) == 0 {
		return status >= 200 && status < 300
	}
	for _, s := range c.ExpectJSONWhenStatus {
		if s == status {
			return true
		}
	}
	return false
}
