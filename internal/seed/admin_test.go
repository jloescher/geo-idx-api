package seed_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quantyralabs/idx-api/internal/auth/password"
	"github.com/quantyralabs/idx-api/internal/seed"
)

func TestAdminUpsert(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	email := "seed-admin-test@example.com"
	_, _ = pool.Exec(ctx, `DELETE FROM users WHERE email = $1`, email)

	id, err := seed.Admin(ctx, pool, email, "test-password-123", "Test Admin")
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("expected user id")
	}

	var hash string
	var isAdmin bool
	err = pool.QueryRow(ctx, `SELECT password, is_admin FROM users WHERE id = $1`, id).Scan(&hash, &isAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if !isAdmin {
		t.Fatal("expected is_admin true")
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("expected argon2id hash, got prefix %q", hash[:12])
	}
	if err := password.Verify("test-password-123", hash); err != nil {
		t.Fatal("password hash mismatch")
	}

	id2, err := seed.Admin(ctx, pool, email, "new-password-456", "Test Admin")
	if err != nil {
		t.Fatal(err)
	}
	if id2 != id {
		t.Fatalf("expected same id %d, got %d", id, id2)
	}

	err = pool.QueryRow(ctx, `SELECT password FROM users WHERE id = $1`, id).Scan(&hash)
	if err != nil {
		t.Fatal(err)
	}
	if err := password.Verify("new-password-456", hash); err != nil {
		t.Fatal("expected password updated on upsert")
	}

	_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
}
