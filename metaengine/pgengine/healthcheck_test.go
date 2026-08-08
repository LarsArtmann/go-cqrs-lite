package pgengine_test

import (
	"context"
	"testing"

	pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestPostgresHealthCheck_Healthy(t *testing.T) {
	t.Parallel()

	eng := mustNewPgEngine(t)

	hc, ok := eng.(metaengine.HealthChecker)
	if !ok {
		t.Fatal("Postgres engine does not implement HealthChecker")
	}

	if err := hc.HealthCheck(context.Background()); err != nil {
		t.Fatalf("HealthCheck on healthy engine: %v", err)
	}
}

func TestPostgresHealthCheck_ClosedDB(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	eng.Close()

	hc := eng.(metaengine.HealthChecker)

	if err := hc.HealthCheck(context.Background()); err == nil {
		t.Fatal("expected HealthCheck to fail on closed DB")
	}
}
