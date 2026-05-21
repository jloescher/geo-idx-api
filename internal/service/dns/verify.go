package dns

import (
	"context"
	"net"
	"strings"
)

// VerifyTXT checks that the expected TXT record exists at the verification host.
func VerifyTXT(ctx context.Context, host, expected string) (bool, error) {
	host = strings.TrimSuffix(strings.TrimSpace(host), ".")
	expected = strings.TrimSpace(expected)
	if host == "" || expected == "" {
		return false, nil
	}
	r := &net.Resolver{}
	records, err := r.LookupTXT(ctx, host)
	if err != nil {
		return false, err
	}
	for _, rec := range records {
		if strings.TrimSpace(rec) == expected {
			return true, nil
		}
	}
	return false, nil
}
