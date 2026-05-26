package password_test

import (
	"strings"
	"testing"

	"github.com/quantyralabs/idx-api/internal/auth/password"
	"golang.org/x/crypto/bcrypt"
)

func TestHashAndVerifyArgon2id(t *testing.T) {
	hash, err := password.Hash("secret-pass", password.DefaultParams)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("expected argon2id prefix, got %q", hash[:20])
	}
	if err := password.Verify("secret-pass", hash); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := password.Verify("wrong", hash); err == nil {
		t.Fatal("expected mismatch")
	}
}

func TestVerifyLegacyBcrypt(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("legacy"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := password.Verify("legacy", string(hash)); err != nil {
		t.Fatal(err)
	}
	if password.NeedsRehash(string(hash)) {
		// expected — bcrypt should be upgraded on next login/seed
	} else {
		t.Fatal("bcrypt should need rehash")
	}
}

func TestNeedsRehash(t *testing.T) {
	h, _ := password.Hash("x", password.DefaultParams)
	if password.NeedsRehash(h) {
		t.Fatal("argon2id should not need rehash")
	}
}
