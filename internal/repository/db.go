package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/quantyralabs/idx-api/internal/config"
)

// DB wraps PostgreSQL connections for repositories.
type DB struct {
	Pool *pgxpool.Pool
	SQLX *sqlx.DB
}

// New opens a pgx pool and sqlx wrapper.
func New(ctx context.Context, cfg config.DBConfig) (*DB, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
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

	sqlxDB, err := sqlx.Connect("pgx", cfg.DSN())
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("sqlx connect: %w", err)
	}

	return &DB{Pool: pool, SQLX: sqlxDB}, nil
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

// Ping checks database connectivity.
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
