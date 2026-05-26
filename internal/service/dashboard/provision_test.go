package dashboard

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/quantyralabs/idx-api/internal/repository"
)

func TestProvisionBundleValidation(t *testing.T) {
	svc := NewProvisionService(nil, nil)
	_, err := svc.ProvisionBundle(context.Background(), 1, ProvisionRequest{})
	if !errors.Is(err, ErrEmptyHostname) {
		t.Fatalf("expected ErrEmptyHostname, got %v", err)
	}
	_, err = svc.ProvisionBundle(context.Background(), 1, ProvisionRequest{Hostname: "staging.example.com"})
	if !errors.Is(err, ErrInvalidHostname) {
		t.Fatalf("expected ErrInvalidHostname, got %v", err)
	}
	_, err = svc.ProvisionBundle(context.Background(), 1, ProvisionRequest{Hostname: "example.com"})
	if !errors.Is(err, ErrNoDatasets) {
		t.Fatalf("expected ErrNoDatasets, got %v", err)
	}
}

func TestProvisionBundleIntegration(t *testing.T) {
	dsn := testDSN(t)
	db, err := repository.NewFromDSN(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tokens := repository.NewTokenRepo(db)
	svc := NewProvisionService(db, tokens)

	host := "test-" + randomSuffix() + ".example.com"
	result, err := svc.ProvisionBundle(context.Background(), 1, ProvisionRequest{
		Hostname: host,
		Datasets: []string{"bridge_stellar", "spark_beaches"},
	})
	if err != nil {
		if errors.Is(err, ErrDuplicateDomain) {
			t.Skip("domain exists from prior run")
		}
		t.Fatal(err)
	}
	if result.ProdDomain.DomainSlug != host {
		t.Fatalf("prod slug: %s", result.ProdDomain.DomainSlug)
	}
	if result.StagingDomain.DomainSlug != "staging."+host {
		t.Fatalf("staging slug: %s", result.StagingDomain.DomainSlug)
	}
	if result.ProductionToken == "" || result.StagingToken == "" {
		t.Fatal("expected plaintext tokens")
	}

	var prodDomainID, stagingDomainID int64
	var stagingParent *int64
	err = db.Pool.QueryRow(context.Background(), `
		SELECT id, parent_domain_id FROM domains WHERE domain_slug = $1
	`, host).Scan(&prodDomainID, &stagingParent)
	if err != nil {
		t.Fatal(err)
	}
	if stagingParent != nil {
		t.Fatalf("production should not have parent: %v", stagingParent)
	}
	err = db.Pool.QueryRow(context.Background(), `
		SELECT id, parent_domain_id FROM domains WHERE domain_slug = $1
	`, "staging."+host).Scan(&stagingDomainID, &stagingParent)
	if err != nil {
		t.Fatal(err)
	}
	if stagingParent == nil || *stagingParent != prodDomainID {
		t.Fatalf("staging parent: %v want %d", stagingParent, prodDomainID)
	}

	var prodTokDomain, stagingTokDomain int64
	err = db.Pool.QueryRow(context.Background(), `
		SELECT domain_id FROM personal_access_tokens WHERE tokenable_id = 1 AND name = 'Production' AND domain_id = $1
	`, prodDomainID).Scan(&prodTokDomain)
	if err != nil {
		t.Fatalf("production token domain_id: %v", err)
	}
	err = db.Pool.QueryRow(context.Background(), `
		SELECT domain_id FROM personal_access_tokens WHERE tokenable_id = 1 AND name = 'Staging' AND domain_id = $1
	`, stagingDomainID).Scan(&stagingTokDomain)
	if err != nil {
		t.Fatalf("staging token domain_id: %v", err)
	}
}

func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	return dsn
}

func randomSuffix() string {
	return repository.RandomTXT()[:8]
}
