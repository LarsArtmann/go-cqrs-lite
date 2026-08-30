package pgengine_test

import (
	"testing"

	pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestPgPlannedOpsMatrix runs the D3 planned-ops parity matrix on live PG
// (testcontainer; skips without a container runtime).
func TestPgPlannedOpsMatrix(t *testing.T) {
	t.Parallel()

	adttest.RunPlannedOpsMatrix(t, []adttest.Factory{
		{
			Name: "postgres",
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
