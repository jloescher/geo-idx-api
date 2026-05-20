package cache

import (
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// ListingsService manages listings_cache (per-domain collection cache).
type ListingsService struct {
	cfg config.Config
	db  *repository.DB
}

func NewListingsService(cfg config.Config, db *repository.DB) *ListingsService {
	return &ListingsService{cfg: cfg, db: db}
}
