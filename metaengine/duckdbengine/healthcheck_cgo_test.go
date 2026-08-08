//go:build cgo

package duckdbengine_test

import (
	"context"
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
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

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	eng.Close()

	hc := eng.(metaengine.HealthChecker)

	if err := hc.HealthCheck(context.Background()); err == nil {
		t.Fatal("expected HealthCheck to fail on closed DB")
	}
}
