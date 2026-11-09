package pgengine_test

import (
	"testing"

	pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestPostgresAggregateMatrix runs the shared aggregate harness (incl. the
// HostileFnRejected SQL-injection pin) against Postgres and the memory engine.
// Skips cleanly when no Postgres is reachable — the CI integration-pg leg
// exercises it with a live server.
func TestPostgresAggregateMatrix(t *testing.T) {
	t.Parallel()

	adttest.RunAggregateMatrix(t, []adttest.Factory{
		{
			Name:   "memory",
			Create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
		},
		{
			Name: "pg",
			Create: func(t *testing.T) metaengine.Engine {
				t.Helper()

				eng, err := pgengine.New(pgDSN(t))
				if err != nil {
					t.Skipf("Postgres not available: %v", err)
				}

				return eng
			},
		},
	})
}
