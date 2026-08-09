//go:build cgo

package duckdbengine_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestDuckDBHealthCheck_Healthy(t *testing.T) {
	t.Parallel()

	eng := mustNewDuckEngine(t)

	hc, ok := eng.(metaengine.HealthChecker)
	if !ok {
		t.Fatal("DuckDB engine does not implement HealthChecker")
	}

	if err := hc.HealthCheck(context.Background()); err != nil {
		t.Fatalf("HealthCheck on healthy engine: %v", err)
	}
}

func TestDuckDBHealthCheck_ClosedDB(t *testing.T) {
	t.Parallel()

	eng := newDuckEngineOrSkip(t)

	eng.Close()

	hc := eng.(metaengine.HealthChecker)

	if err := hc.HealthCheck(context.Background()); err == nil {
		t.Fatal("expected HealthCheck to fail on closed DB")
	}
}
