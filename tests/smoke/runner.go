//go:build smoke

package smoke

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// RunAll executes every loaded case and returns a reporter with results.
func RunAll(t *testing.T, repoRoot string, cfg Config, cases []Case, dbFix DBFixtures) *Reporter {
	t.Helper()
	reporter := NewReporter(repoRoot, cfg)
	client := NewClient(cfg)

	if cfg.DomainSlug == "" || cfg.DomainSlug == "example.com" {
		cfg.DomainSlug = dbFix.DomainSlug
	}
	if !cfg.HasAuth() && cfg.AdminEmail != "" && cfg.AdminPassword != "" {
		token, err := client.Login(cfg.AdminEmail, cfg.AdminPassword)
		if err == nil && token != "" {
			client.SetToken(token)
			cfg.BearerToken = token
		}
	}
	if !cfg.HasAuth() {
		reporter.Warn()
		t.Logf("WARN: no bearer token (set YAAK_BEARER_TOKEN or ADMIN_SEED_*); MLS routes expect 401")
	}

	vars := templateVars{
		ListingKey:       dbFix.ListingKey,
		MLSListingID:     dbFix.MLSListingID,
		PhotoID:          dbFix.PhotoID,
		AgentID:          dbFix.AgentID,
		OfficeID:         dbFix.OfficeID,
		OpenHouseID:      dbFix.OpenHouseID,
		MemberKey:        dbFix.MemberKey,
		ResoOfficeKey:    dbFix.ResoOfficeKey,
		ResoOpenHouseKey: dbFix.ResoOpenHouseKey,
		Dataset:          cfg.Dataset,
		DomainSlug:       cfg.DomainSlug,
		BBox:             cfg.BBox,
		AdminEmail:       cfg.AdminEmail,
		AdminPassword:    cfg.AdminPassword,
	}

	examplesDir := filepath.Join(repoRoot, "docs", "client-examples")
	for _, raw := range cases {
		c := raw.resolved(vars)
		if c.Skip != "" {
			reporter.Skip()
			t.Logf("SKIP %s — %s", c.Name, c.Skip)
			continue
		}

		auth := c.wantsAuth(cfg)
		resp, err := client.Do(c.Method, c.Path, c.Query, c.Headers, []byte(c.Body), auth)
		if err != nil {
			reporter.Fail()
			recordFailure(reporter, cfg, c, resp, err.Error(), []string{err.Error()}, nil)
			t.Errorf("%s: request failed: %v", c.Name, err)
			continue
		}

		allowed := c.allowedStatuses(cfg)
		if !statusAllowed(resp.StatusCode, allowed) {
			reporter.Fail()
			msg := fmt.Sprintf("expected status %v, got %d", allowed, resp.StatusCode)
			recordFailure(reporter, cfg, c, resp, msg, nil, allowed)
			t.Errorf("%s: %s body=%s", c.Name, msg, truncateBody(string(resp.Body), 200))
			continue
		}

		var assertFailures []string
		if c.shouldValidateJSON(resp.StatusCode) {
			if c.ExpectJSONArray && !isJSONArray(resp.Body) {
				assertFailures = append(assertFailures, "expected top-level JSON array")
			}
			if c.ExpectJSONObject && !isJSONObject(resp.Body) {
				assertFailures = append(assertFailures, "expected top-level JSON object")
			}
			assertFailures = append(assertFailures, checkJSONExpectations(resp.Body, c.ExpectJSON)...)
			assertFailures = append(assertFailures, checkJSONEquals(resp.Body, c.ExpectJSONEquals)...)
		}
		if c.ExpectContentTypePrefix != "" && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if !strings.HasPrefix(resp.ContentType, c.ExpectContentTypePrefix) {
				assertFailures = append(assertFailures, fmt.Sprintf("content-type %q does not start with %q", resp.ContentType, c.ExpectContentTypePrefix))
			}
		}
		if c.ExpectMinBodyBytes > 0 && resp.StatusCode >= 200 && resp.StatusCode < 300 && len(resp.Body) < c.ExpectMinBodyBytes {
			assertFailures = append(assertFailures, fmt.Sprintf("body length %d < min %d", len(resp.Body), c.ExpectMinBodyBytes))
		}

		if len(assertFailures) > 0 {
			reporter.Fail()
			recordFailure(reporter, cfg, c, resp, strings.Join(assertFailures, "; "), assertFailures, allowed)
			t.Errorf("%s: %s", c.Name, strings.Join(assertFailures, "; "))
			continue
		}

		if statusWarn(resp.StatusCode, c.WarnOnStatus) {
			reporter.Warn()
			t.Logf("WARN [%d] %s %s %s", resp.StatusCode, c.Method, c.Path, c.Name)
		}

		reporter.Pass()
		t.Logf("PASS [%d] %s %s %s", resp.StatusCode, c.Method, c.Path, c.Name)

		if c.ExportAs != "" && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if err := exportClientExample(examplesDir, c, resp, cfg, client); err != nil {
				t.Logf("export %s: %v", c.ExportAs, err)
			}
		}
	}

	if err := reporter.Write(); err != nil {
		t.Logf("write report: %v", err)
	}
	reporter.PrintAIFeedbackBlock()
	return reporter
}

func statusAllowed(code int, allowed []int) bool {
	for _, s := range allowed {
		if code == s {
			return true
		}
	}
	return false
}

func statusWarn(code int, warn []int) bool {
	for _, s := range warn {
		if code == s {
			return true
		}
	}
	return false
}

func recordFailure(reporter *Reporter, cfg Config, c Case, resp HTTPResponse, diagnosis string, details []string, allowed []int) {
	headers := map[string]string{}
	for k, v := range c.Headers {
		headers[k] = v
	}
	if c.wantsAuth(cfg) && cfg.HasAuth() {
		headers["Authorization"] = "Bearer …"
		headers["X-Domain-Slug"] = cfg.DomainSlug
	}
	entry := FailureEntry{
		Case:     c.Name,
		Endpoint: fmt.Sprintf("%s %s", c.Method, resp.URL),
		Request: RequestDetail{
			Method:  c.Method,
			URL:     resp.URL,
			Headers: headers,
			Body:    c.Body,
			Curl:    buildCurl(c.Method, resp.URL, headers, c.Body),
		},
		Expected: ExpectedDetail{
			Status:    allowed,
			JSONPaths: c.ExpectJSON,
		},
		Actual: ActualDetail{
			Status:      resp.StatusCode,
			ContentType: resp.ContentType,
			BodyPreview: truncateBody(string(resp.Body), 500),
			JSONPaths:   collectJSONPaths(resp.Body, ""),
		},
		Diagnosis:  diagnosis,
		NextJSHint: c.NextJSHint,
	}
	if len(details) > 0 && entry.Diagnosis == "" {
		entry.Diagnosis = strings.Join(details, "; ")
	}
	reporter.AddFailure(entry)
}

type clientExample struct {
	Description string            `json:"description"`
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Status      int               `json:"status"`
	Headers     map[string]string `json:"headers"`
	Body        json.RawMessage   `json:"body,omitempty"`
	NextJS      nextJSExample     `json:"nextjs_fetch_example"`
}

type nextJSExample struct {
	Note    string `json:"note"`
	Snippet string `json:"snippet"`
}

func exportClientExample(dir string, c Case, resp HTTPResponse, cfg Config, client *Client) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	for k, v := range c.Headers {
		headers[k] = v
	}
	if c.wantsAuth(cfg) {
		headers["Authorization"] = "Bearer ${process.env.IDX_API_TOKEN}"
		headers["X-Domain-Slug"] = "${process.env.IDX_DOMAIN_SLUG}"
	}
	url := resp.URL
	ex := clientExample{
		Description: c.Name,
		Method:      c.Method,
		URL:         url,
		Status:      resp.StatusCode,
		Headers:     headers,
		Body:        c.Body,
		NextJS: nextJSExample{
			Note:    "Use in a Next.js Server Action or Route Handler; never expose the PAT to the browser.",
			Snippet: buildNextJSSnippet(c.Method, url, headers, c.Body),
		},
	}
	b, err := json.MarshalIndent(ex, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, c.ExportAs), b, 0o644)
}

func buildNextJSSnippet(method, url string, headers map[string]string, body json.RawMessage) string {
	var b strings.Builder
	b.WriteString("const res = await fetch(")
	b.WriteString(fmt.Sprintf("%q", url))
	b.WriteString(", {\n  method: ")
	b.WriteString(fmt.Sprintf("%q", method))
	b.WriteString(",\n  headers: ")
	h, _ := json.MarshalIndent(headers, "  ", "  ")
	b.Write(h)
	if len(body) > 0 && method != http.MethodGet {
		b.WriteString(",\n  body: JSON.stringify(")
		b.WriteString(string(body))
		b.WriteString("),\n")
	} else {
		b.WriteString(",\n")
	}
	b.WriteString("});\nif (!res.ok) throw new Error(`API ${res.status}`);\nconst data = await res.json();")
	return b.String()
}
