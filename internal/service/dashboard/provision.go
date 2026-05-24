package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/quantyralabs/idx-api/internal/repository"
)

// Simultaneous domain + prod/staging token creation + multi-dataset support + live monitoring =
// faster domain onboarding → more pilot domains live faster → higher organic traffic and lead
// volume while giving ops full visibility into data freshness (critical for MLS GRID compliance and conversion).

// ProvisionRequest is input for atomic prod/staging domain + token bundle.
type ProvisionRequest struct {
	Hostname string
	Datasets []string
}

// CreatedDomain holds a newly inserted domain row summary.
type CreatedDomain struct {
	ID                   int64
	DomainSlug           string
	TXTVerificationName  string
	TXTVerificationValue string
}

// ProvisionResult is returned after a successful bundle.
type ProvisionResult struct {
	ProdDomain        CreatedDomain
	StagingDomain     CreatedDomain
	ProductionToken   string
	StagingToken      string
	AllowedDatasets   []string
	DefaultMLSDataset string
}

var (
	ErrEmptyHostname   = errors.New("hostname is required")
	ErrInvalidHostname = errors.New("hostname cannot start with staging.")
	ErrNoDatasets      = errors.New("select at least one MLS dataset")
	ErrDuplicateDomain = errors.New("domain already registered")
)

// ProvisionService creates prod + staging domains and matching API tokens atomically.
type ProvisionService struct {
	db     *repository.DB
	tokens *repository.TokenRepo
}

// NewProvisionService wires domain provisioning dependencies.
func NewProvisionService(db *repository.DB, tokens *repository.TokenRepo) *ProvisionService {
	return &ProvisionService{db: db, tokens: tokens}
}

// ProvisionBundle creates production domain, staging domain, Production token, and Staging token in one transaction.
// Revenue impact: one-click onboarding removes friction before the first IDX map call and accelerates lead capture.
func (s *ProvisionService) ProvisionBundle(ctx context.Context, userID int64, req ProvisionRequest) (*ProvisionResult, error) {
	host := strings.ToLower(strings.TrimSpace(req.Hostname))
	if host == "" {
		return nil, ErrEmptyHostname
	}
	if strings.HasPrefix(host, "staging.") {
		return nil, ErrInvalidHostname
	}
	if len(req.Datasets) == 0 {
		return nil, ErrNoDatasets
	}
	stagingHost := "staging." + host
	allowedJSON, err := json.Marshal(req.Datasets)
	if err != nil {
		return nil, err
	}
	defaultDS := req.Datasets[0]

	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	prodTXT := repository.RandomTXT()
	stagingTXT := repository.RandomTXT()

	prodID, err := repository.CreateDomain(ctx, tx, repository.DomainCreateParams{
		UserID:               userID,
		DomainSlug:           host,
		MLSDataset:           defaultDS,
		AllowedMLSDatasets:   allowedJSON,
		TXTVerificationName:  "_quantyra-verify." + host,
		TXTVerificationValue: prodTXT,
	})
	if err != nil {
		if repository.IsUniqueViolation(err) {
			return nil, ErrDuplicateDomain
		}
		return nil, err
	}

	stagingID, err := repository.CreateDomain(ctx, tx, repository.DomainCreateParams{
		UserID:               userID,
		DomainSlug:           stagingHost,
		MLSDataset:           defaultDS,
		AllowedMLSDatasets:   allowedJSON,
		TXTVerificationName:  "_quantyra-verify." + stagingHost,
		TXTVerificationValue: stagingTXT,
	})
	if err != nil {
		if repository.IsUniqueViolation(err) {
			return nil, fmt.Errorf("staging domain already registered: %w", ErrDuplicateDomain)
		}
		return nil, err
	}

	prodPlain, err := s.tokens.CreateWithTx(ctx, tx, userID, "Production", []string{"idx:full"})
	if err != nil {
		return nil, err
	}
	stagingPlain, err := s.tokens.CreateWithTx(ctx, tx, userID, "Staging", []string{"idx:full"})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &ProvisionResult{
		ProdDomain: CreatedDomain{
			ID:                   prodID,
			DomainSlug:           host,
			TXTVerificationName:  "_quantyra-verify." + host,
			TXTVerificationValue: prodTXT,
		},
		StagingDomain: CreatedDomain{
			ID:                   stagingID,
			DomainSlug:           stagingHost,
			TXTVerificationName:  "_quantyra-verify." + stagingHost,
			TXTVerificationValue: stagingTXT,
		},
		ProductionToken:   prodPlain,
		StagingToken:      stagingPlain,
		AllowedDatasets:   req.Datasets,
		DefaultMLSDataset: defaultDS,
	}, nil
}
