.PHONY: build test fmt lint run-api run-worker run-scheduler migrate migrate-install seed-admin gis-enqueue-parcels gis-enqueue-zips openapi-sync

# Copy OpenAPI source into the API binary embed path (after editing docs/yaak-api-collection.json).
openapi-sync:
	@mkdir -p internal/openapi/spec
	cp docs/yaak-api-collection.json internal/openapi/spec/openapi.json

GOFLAGS ?= -mod=mod
GOOSE_PKG := github.com/pressly/goose/v3/cmd/goose@v3.24.3

build: openapi-sync
	GOFLAGS=$(GOFLAGS) go build -o bin/api ./cmd/api
	GOFLAGS=$(GOFLAGS) go build -o bin/worker ./cmd/worker
	GOFLAGS=$(GOFLAGS) go build -o bin/scheduler ./cmd/scheduler

test:
	GOFLAGS=$(GOFLAGS) go test ./...

fmt:
	gofmt -w cmd internal

lint:
	golangci-lint run ./...

run-api:
	GOFLAGS=$(GOFLAGS) go run ./cmd/api

run-worker:
	GOFLAGS=$(GOFLAGS) go run ./cmd/worker

run-scheduler:
	GOFLAGS=$(GOFLAGS) go run ./cmd/scheduler

# Bootstrap admin from ADMIN_SEED_EMAIL / ADMIN_SEED_PASSWORD in .env
seed-admin:
	GOFLAGS=$(GOFLAGS) go run ./cmd/seed

# Enqueue GIS backfill jobs into the PostgreSQL queue (worker must be running).
gis-enqueue-parcels:
	GOFLAGS=$(GOFLAGS) go run ./cmd/gis-enqueue -job parcels

gis-enqueue-zips:
	GOFLAGS=$(GOFLAGS) go run ./cmd/gis-enqueue -job zips

# Run migrations without a global goose binary (uses go run).
# Example: GOOSE_DBSTRING='postgres://postgres:@127.0.0.1:5432/idx_api?sslmode=disable' make migrate
migrate:
	@test -n "$$GOOSE_DBSTRING" || (echo "Set GOOSE_DBSTRING, e.g. postgres://postgres:@127.0.0.1:5432/idx_api?sslmode=disable" && exit 1)
	GOOSE_DRIVER=postgres GOOSE_DBSTRING="$$GOOSE_DBSTRING" GOFLAGS=$(GOFLAGS) go run $(GOOSE_PKG) -dir migrations up

# Optional: install goose on PATH ($(go env GOPATH)/bin must be in PATH)
migrate-install:
	go install $(GOOSE_PKG)
