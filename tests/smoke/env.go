//go:build smoke

package smoke

import (
	"os"
	"strings"
)

// Config holds smoke test runtime settings from environment variables.
type Config struct {
	BaseURL     string
	BearerToken string
	DomainSlug  string
	Dataset     string
	BBox        string
	DBDSN       string
	SessionCookie string

	AdminEmail    string
	AdminPassword string
}

func LoadConfig() Config {
	cfg := Config{
		BaseURL:       envOr("YAAK_BASE_URL", "https://idx.quantyralabs.cc"),
		BearerToken:   strings.TrimSpace(os.Getenv("YAAK_BEARER_TOKEN")),
		DomainSlug:    strings.TrimSpace(os.Getenv("YAAK_DOMAIN_SLUG")),
		Dataset:       envOr("YAAK_DATASET", "stellar"),
		BBox:          envOr("YAAK_BBOX", "-82.8,27.9,-82.6,28.0"),
		DBDSN:         strings.TrimSpace(os.Getenv("GOOSE_DBSTRING")),
		SessionCookie: strings.TrimSpace(os.Getenv("YAAK_SESSION_COOKIE")),
		AdminEmail:    strings.TrimSpace(os.Getenv("ADMIN_SEED_EMAIL")),
		AdminPassword: strings.TrimSpace(os.Getenv("ADMIN_SEED_PASSWORD")),
	}
	if os.Getenv("YAAK_USE_APP_URL") == "1" {
		if appURL := strings.TrimSpace(os.Getenv("APP_URL")); appURL != "" {
			cfg.BaseURL = appURL
		}
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.DomainSlug == "" {
		cfg.DomainSlug = "example.com"
	}
	return cfg
}

func (c Config) HasAuth() bool {
	return c.BearerToken != ""
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
