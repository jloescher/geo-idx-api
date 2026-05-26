package repository

import (
	"crypto/rand"
	"encoding/hex"
)

// RandomTXT returns a DNS TXT verification value.
func RandomTXT() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
