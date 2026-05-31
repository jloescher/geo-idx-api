package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// FingerprintRequest hashes method, path, sorted query (excluding internal domain), and body.
func FingerprintRequest(c *fiber.Ctx, upstreamPath string) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s\n%s\n", c.Method(), upstreamPath)
	writeQueryFingerprint(h, c)
	if len(c.Body()) > 0 {
		h.Write(c.Body())
	}
	return hex.EncodeToString(h.Sum(nil))
}

// LogicalFingerprint hashes the inbound IDX request without upstream URL (for multi-leg cache keys).
func LogicalFingerprint(c *fiber.Ctx) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s\n%s\n", c.Method(), c.Path())
	writeQueryFingerprint(h, c)
	if len(c.Body()) > 0 {
		h.Write(c.Body())
	}
	return hex.EncodeToString(h.Sum(nil))
}

// FingerprintWithLeg appends an upstream leg suffix to a logical fingerprint.
func FingerprintWithLeg(c *fiber.Ctx, leg string) string {
	return LogicalFingerprint(c) + ":" + leg
}

func writeQueryFingerprint(h hashWriter, c *fiber.Ctx) {
	keys := make([]string, 0, len(c.Queries()))
	for k := range c.Queries() {
		if strings.EqualFold(k, "domain") {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		_, _ = fmt.Fprintf(h, "%s=%s\n", k, c.Query(k))
	}
}

type hashWriter interface {
	Write(p []byte) (n int, err error)
}

// WebPartition keys web proxy responses per domain, feed, and audit route type.
func WebPartition(domainSlug, feedCode, auditType string) string {
	return fmt.Sprintf("%s:%s:web:%s", domainSlug, feedCode, auditType)
}

// ResoPartition keys RESO proxy responses.
func ResoPartition(domainSlug, feedCode, entity string) string {
	return fmt.Sprintf("%s:%s:reso:%s", domainSlug, feedCode, entity)
}

// SearchPartition keys hybrid search live-leg responses.
func SearchPartition(domainSlug, feedCode string) string {
	return fmt.Sprintf("%s:%s:search", domainSlug, feedCode)
}

// LookupPartition uses longer TTL for lookup routes.
func LookupPartition(domainSlug, feedCode string) string {
	return fmt.Sprintf("%s:%s:lookup", domainSlug, feedCode)
}
