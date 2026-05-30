//go:build smoke

package smoke_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/quantyralabs/idx-api/tests/smoke"
)

func TestSmokeAll(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	casesDir := filepath.Join(repoRoot, "tests", "smoke", "cases")

	cfg := smoke.LoadConfig()
	t.Logf("Base URL: %s", cfg.BaseURL)
	t.Logf("Domain:   %s", cfg.DomainSlug)
	t.Logf("Dataset:  %s", cfg.Dataset)

	dbFix, err := smoke.ResolveDBFixtures(cfg)
	if err != nil {
		t.Logf("DB fixtures: %v", err)
	}
	if cfg.DomainSlug == "" || cfg.DomainSlug == "example.com" {
		cfg.DomainSlug = dbFix.DomainSlug
	}
	t.Logf("Fixture listing_key: %s", dbFix.ListingKey)

	cases, err := smoke.LoadCases(casesDir)
	if err != nil {
		t.Fatalf("load cases: %v", err)
	}
	if cfg.AdminEmail != "" && cfg.AdminPassword != "" {
		loginBody, err := json.Marshal(map[string]string{
			"email":    cfg.AdminEmail,
			"password": cfg.AdminPassword,
		})
		if err != nil {
			t.Fatalf("marshal login body: %v", err)
		}
		cases = append(cases, smoke.Case{
			Name:   "auth_token_login",
			Method: "POST",
			Path:   "/api/auth/token",
			Auth:   boolPtr(false),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body:         loginBody,
			ExpectStatus: []int{200},
			ExpectJSON: map[string]string{
				"token":     "string",
				"abilities": "array",
			},
			ExportAs:   "auth-token-login.json",
			NextJSHint: "Obtain PAT server-side only; store in env, not client bundle.",
		})
	}
	if len(cases) == 0 {
		t.Fatal("no smoke cases loaded")
	}

	reporter := smoke.RunAll(t, repoRoot, cfg, cases, dbFix)
	t.Logf("Summary: pass=%d fail=%d skip=%d warn=%d",
		reporter.Summary().Pass, reporter.Summary().Fail, reporter.Summary().Skip, reporter.Summary().Warn)

	if reporter.Summary().Fail > 0 {
		reportPath := filepath.Join(repoRoot, "tests", "smoke", "reports", "latest.json")
		t.Fatalf("%d smoke test(s) failed; see %s", reporter.Summary().Fail, reportPath)
	}
}

func boolPtr(v bool) *bool { return &v }

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
