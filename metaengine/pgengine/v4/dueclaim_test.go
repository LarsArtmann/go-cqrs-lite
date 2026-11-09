package pgengine_test

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestPostgresDueClaims pins the engine's ADR-0142 capabilities (claimkit
// SQL runtime: CTE FOR UPDATE SKIP LOCKED claims, PK dedup upserts) to the
// shared conformance contract. Skips when Postgres is unavailable; the
// integration leg (PG_MODULES="metaengine/pgengine" nix run .#integration-pg)
// runs it against live Postgres.
func TestPostgresDueClaims(t *testing.T) {
	t.Parallel()

	adttest.AssertDueClaimer(t, []adttest.Factory{
		{
			Name:   "postgres",
			Create: func(t *testing.T) metaengine.Engine { return newPgEngineOrSkip(t) },
		},
	})
	adttest.AssertDedupStore(t, []adttest.Factory{
		{
			Name:   "postgres",
			Create: func(t *testing.T) metaengine.Engine { return newPgEngineOrSkip(t) },
		},
	})
	adttest.AssertFactSink(t, []adttest.Factory{
		{
			Name:   "postgres",
			Create: func(t *testing.T) metaengine.Engine { return newPgEngineOrSkip(t) },
		},
	})
}

func TestPostgresDueClaims_CapabilitySurface(t *testing.T) {
	t.Parallel()

	eng := newPgEngineOrSkip(t)

	if !metaengine.SupportsDueClaims(eng) || !metaengine.SupportsDedup(eng) {
		t.Fatal("postgres engine must satisfy DueClaimer + DedupStore after claimkit wiring")
	}

	if _, ok := eng.(metaengine.FactSink); !ok {
		t.Fatal("postgres engine must satisfy FactSink after claimkit wiring")
	}

	if eng.Profile().IsDegraded(metaengine.ADTDueClaim) {
		t.Fatal("postgres claims are native (CTE SKIP LOCKED), not degraded")
	}
}
