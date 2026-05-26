package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/db"
)

// DB wraps PostgreSQL connections for repositories.
type DB struct {
	Pool     *pgxpool.Pool // RW — writes and fallback reads
	SQLX     *sqlx.DB      // RW — SQLX mutations and legacy write paths
	Selector *db.ReplicaSelector
}

// New opens a pgx pool and sqlx wrapper using the configured RW DSN.
func New(ctx context.Context, cfg config.DBConfig) (*DB, error) {
	return NewFromDSN(ctx, cfg.RWDSNOrDefault())
}

// NewFromDSN opens pools from an explicit DSN (used for tests and DB_RW_DSN).
func NewFromDSN(ctx context.Context, dsn string) (*DB, error) {
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	sqlxDB, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("sqlx connect: %w", err)
	}

	return &DB{Pool: pool, SQLX: sqlxDB}, nil
}

// ReadPool returns the lowest-latency replica pool when a selector is configured,
// otherwise the primary RW pool.
func (db *DB) ReadPool(ctx context.Context) (*pgxpool.Pool, error) {
	if db.Selector != nil {
		return db.Selector.GetBestReplica(ctx)
	}
	return db.Pool, nil
}

// Close releases connections.
func (db *DB) Close() {
	if db.SQLX != nil {
		_ = db.SQLX.Close()
	}
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// Ping checks database connectivity on the RW pool.
func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// PostGISVersion returns PostGIS version string (empty if unavailable).
func (db *DB) PostGISVersion(ctx context.Context) (string, error) {
	var ver string
	err := db.Pool.QueryRow(ctx, `SELECT PostGIS_Version()`).Scan(&ver)
	if err != nil {
		return "", err
	}
	return ver, nil
}
