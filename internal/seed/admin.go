package seed

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quantyralabs/idx-api/internal/auth/password"
)

// Admin upserts the bootstrap admin user (Laravel AdminUserSeeder parity).
func Admin(ctx context.Context, pool *pgxpool.Pool, email, plainPassword, name string) (int64, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	plainPassword = strings.TrimSpace(plainPassword)
	name = strings.TrimSpace(name)
	if email == "" || plainPassword == "" {
		return 0, fmt.Errorf("ADMIN_SEED_EMAIL and ADMIN_SEED_PASSWORD are required")
	}
	if name == "" {
		name = "Quantyra Admin"
	}

	hash, err := password.Hash(plainPassword, password.DefaultParams)
	if err != nil {
		return 0, err
	}

	var id int64
	err = pool.QueryRow(ctx, `
		INSERT INTO users (name, email, is_admin, email_verified_at, password, created_at, updated_at)
		VALUES ($1, $2, TRUE, NOW(), $3, NOW(), NOW())
		ON CONFLICT (email) DO UPDATE SET
			name = EXCLUDED.name,
			password = EXCLUDED.password,
			is_admin = TRUE,
			email_verified_at = COALESCE(users.email_verified_at, NOW()),
			updated_at = NOW()
		RETURNING id
	`, name, email, hash).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}
