package sync

import (
	"testing"

	"github.com/quantyralabs/idx-api/internal/config"
)

func TestMirrorRollingMonthsDescription(t *testing.T) {
	if got := MirrorRollingMonthsDescription(config.Config{MLS: config.MLSConfig{LocalMirrorRollingMonths: 0}}); got != "all-time" {
		t.Fatalf("got %q", got)
	}
	if got := MirrorRollingMonthsDescription(config.Config{MLS: config.MLSConfig{LocalMirrorRollingMonths: 3}}); got != "3-month" {
		t.Fatalf("got %q", got)
	}
}
