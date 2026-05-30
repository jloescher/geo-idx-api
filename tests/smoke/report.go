//go:build smoke

package smoke

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// FailureReport is written to tests/smoke/reports/latest.json for AI debugging.
type FailureReport struct {
	GeneratedAt string         `json:"generated_at"`
	BaseURL     string         `json:"base_url"`
	Summary     ReportSummary  `json:"summary"`
	Failures    []FailureEntry `json:"failures"`
}

type ReportSummary struct {
	Pass int `json:"pass"`
	Fail int `json:"fail"`
	Skip int `json:"skip"`
	Warn int `json:"warn"`
}

type FailureEntry struct {
	Case       string         `json:"case"`
	Endpoint   string         `json:"endpoint"`
	Request    RequestDetail  `json:"request"`
	Expected   ExpectedDetail `json:"expected"`
	Actual     ActualDetail   `json:"actual"`
	Diagnosis  string         `json:"diagnosis"`
	NextJSHint string         `json:"nextjs_hint,omitempty"`
}

type RequestDetail struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    json.RawMessage   `json:"body,omitempty"`
	Curl    string            `json:"curl"`
}

type ExpectedDetail struct {
	Status    []int             `json:"status"`
	JSONPaths map[string]string `json:"json_paths,omitempty"`
}

type ActualDetail struct {
	Status      int            `json:"status"`
	ContentType string         `json:"content_type,omitempty"`
	BodyPreview string         `json:"body_preview"`
	JSONPaths   map[string]any `json:"json_paths,omitempty"`
}

type Reporter struct {
	path     string
	cfg      Config
	summary  ReportSummary
	failures []FailureEntry
}

func NewReporter(repoRoot string, cfg Config) *Reporter {
	return &Reporter{
		path: filepath.Join(repoRoot, "tests", "smoke", "reports", "latest.json"),
		cfg:  cfg,
	}
}

func (r *Reporter) Pass() { r.summary.Pass++ }
func (r *Reporter) Fail() { r.summary.Fail++ }
func (r *Reporter) Skip() { r.summary.Skip++ }
func (r *Reporter) Warn() { r.summary.Warn++ }

func (r *Reporter) Summary() ReportSummary {
	return r.summary
}

func (r *Reporter) AddFailure(entry FailureEntry) {
	r.failures = append(r.failures, entry)
}

func (r *Reporter) Write() error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}
	report := FailureReport{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		BaseURL:     r.cfg.BaseURL,
		Summary:     r.summary,
		Failures:    r.failures,
	}
	if report.Failures == nil {
		report.Failures = []FailureEntry{}
	}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, b, 0o644)
}

func (r *Reporter) PrintAIFeedbackBlock() {
	if len(r.failures) == 0 {
		return
	}
	fmt.Println("")
	fmt.Println("=== Cursor AI feedback block (copy below) ===")
	fmt.Println("Production API smoke tests failed. Fix the API or client request using this evidence:")
	for i, f := range r.failures {
		fmt.Printf("\n--- Failure %d: %s ---\n", i+1, f.Case)
		fmt.Printf("Endpoint: %s\n", f.Endpoint)
		fmt.Printf("Diagnosis: %s\n", f.Diagnosis)
		if f.NextJSHint != "" {
			fmt.Printf("NextJS hint: %s\n", f.NextJSHint)
		}
		fmt.Printf("Expected status: %v\n", f.Expected.Status)
		fmt.Printf("Actual status: %d\n", f.Actual.Status)
		fmt.Printf("Curl:\n%s\n", f.Request.Curl)
		fmt.Printf("Body preview: %s\n", f.Actual.BodyPreview)
	}
	fmt.Println("=== end feedback block ===")
}

func buildCurl(method, url string, headers map[string]string, body []byte) string {
	var b strings.Builder
	b.WriteString("curl -sS -X ")
	b.WriteString(method)
	b.WriteString(" '")
	b.WriteString(url)
	b.WriteString("'")
	for k, v := range headers {
		if strings.EqualFold(k, "Authorization") && strings.HasPrefix(v, "Bearer ") {
			b.WriteString(" -H 'Authorization: Bearer …'")
			continue
		}
		b.WriteString(fmt.Sprintf(" -H %q: %s", k, v))
	}
	if len(body) > 0 {
		b.WriteString(" -d ")
		b.WriteString(strconv.Quote(string(body)))
	}
	return b.String()
}

func truncateBody(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
