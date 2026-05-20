// Package password hashes and verifies passwords with Argon2id (PHC format).
// Legacy bcrypt hashes ($2y$ / $2a$ / $2b$) are accepted until re-seeded or password reset.
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// Params match Laravel config/hashing.php argon defaults (memory KiB, time, threads).
type Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultParams is Laravel-aligned Argon2id for new hashes.
var DefaultParams = Params{
	Memory:      65536,
	Iterations:  4,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

var (
	ErrInvalidHash = errors.New("invalid password hash")
	ErrMismatch    = errors.New("password does not match hash")
)

// Hash returns an Argon2id PHC string: $argon2id$v=19$m=...,t=...,p=...$salt$hash
func Hash(plain string, p Params) (string, error) {
	if p.SaltLength == 0 {
		p = DefaultParams
	}
	salt := make([]byte, p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	digest := argon2.IDKey([]byte(plain), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(digest)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Iterations, p.Parallelism, b64Salt, b64Hash), nil
}

// Verify checks plain against a stored Argon2id or legacy bcrypt hash.
func Verify(plain, encoded string) error {
	encoded = strings.TrimSpace(encoded)
	if strings.HasPrefix(encoded, "$argon2") {
		return verifyArgon2(plain, encoded)
	}
	if strings.HasPrefix(encoded, "$2a$") || strings.HasPrefix(encoded, "$2b$") || strings.HasPrefix(encoded, "$2y$") {
		if err := bcrypt.CompareHashAndPassword([]byte(encoded), []byte(plain)); err != nil {
			return ErrMismatch
		}
		return nil
	}
	return ErrInvalidHash
}

func verifyArgon2(plain, encoded string) error {
	p, salt, hash, err := decodeArgon2(encoded)
	if err != nil {
		return err
	}
	other := argon2.IDKey([]byte(plain), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)
	if subtle.ConstantTimeCompare(hash, other) != 1 {
		return ErrMismatch
	}
	return nil
}

func decodeArgon2(encoded string) (Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return Params{}, nil, nil, ErrInvalidHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return Params{}, nil, nil, ErrInvalidHash
	}
	var memory, iterations uint64
	var parallelism uint64
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	if parallelism > 255 {
		return Params{}, nil, nil, ErrInvalidHash
	}
	salt, err := decodeBase64(parts[4])
	if err != nil {
		return Params{}, nil, nil, err
	}
	hash, err := decodeBase64(parts[5])
	if err != nil {
		return Params{}, nil, nil, err
	}
	return Params{
		Memory:      uint32(memory),
		Iterations:  uint32(iterations),
		Parallelism: uint8(parallelism),
		KeyLength:   uint32(len(hash)),
	}, salt, hash, nil
}

func decodeBase64(s string) ([]byte, error) {
	b, err := base64.RawStdEncoding.DecodeString(s)
	if err == nil {
		return b, nil
	}
	// PHP/Laravel password_hash may include padding.
	return base64.StdEncoding.DecodeString(s)
}

// NeedsRehash reports whether stored hash should be upgraded to Argon2id.
func NeedsRehash(encoded string) bool {
	return !strings.HasPrefix(strings.TrimSpace(encoded), "$argon2id$")
}

// FormatLabel returns a short label for logs (never log the hash itself).
func FormatLabel(encoded string) string {
	encoded = strings.TrimSpace(encoded)
	switch {
	case strings.HasPrefix(encoded, "$argon2id$"):
		return "argon2id"
	case strings.HasPrefix(encoded, "$2"):
		return "bcrypt"
	default:
		return "unknown"
	}
}
