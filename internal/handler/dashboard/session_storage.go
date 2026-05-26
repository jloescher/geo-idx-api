package dashboard

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgSessionStorage persists Fiber dashboard sessions in PostgreSQL (multi-instance safe).
type pgSessionStorage struct {
	pool *pgxpool.Pool
}

func newPGSessionStorage(pool *pgxpool.Pool) *pgSessionStorage {
	return &pgSessionStorage{pool: pool}
}

func (s *pgSessionStorage) Get(key string) ([]byte, error) {
	if key == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var payload []byte
	err := s.pool.QueryRow(ctx, `
		SELECT payload FROM dashboard_sessions
		WHERE id = $1 AND expires_at > NOW()
	`, key).Scan(&payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return payload, err
}

func (s *pgSessionStorage) Set(key string, val []byte, exp time.Duration) error {
	if key == "" || len(val) == 0 {
		return nil
	}
	if exp <= 0 {
		exp = 24 * time.Hour
	}
	expires := time.Now().Add(exp)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `
		INSERT INTO dashboard_sessions (id, payload, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE
		SET payload = EXCLUDED.payload, expires_at = EXCLUDED.expires_at
	`, key, val, expires)
	return err
}

func (s *pgSessionStorage) Delete(key string) error {
	if key == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `DELETE FROM dashboard_sessions WHERE id = $1`, key)
	return err
}

func (s *pgSessionStorage) Reset() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `DELETE FROM dashboard_sessions`)
	return err
}

func (s *pgSessionStorage) Close() error {
	return nil
}
