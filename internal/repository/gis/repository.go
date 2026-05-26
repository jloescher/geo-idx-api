package gisrepo

import (
	"github.com/quantyralabs/idx-api/internal/repository"
)

// Repository wraps PostGIS GIS table access.
type Repository struct {
	db *repository.DB
}

// New creates a GIS repository.
func New(db *repository.DB) *Repository {
	return &Repository{db: db}
}
