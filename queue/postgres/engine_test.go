package postgres_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
	"github.com/larsartmann/go-cqrs-lite/queue/postgres/v4"
)

// queueDSN resolves the test DSN; skips unless POSTGRES_TEST_DSN is set
// (the integration leg and ephemeral-pg.sh export it).
func queueDSN(t *testing.T) string {
	t.Helper()

	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN not set — run via the integration leg")
	}

	return dsn
}

// newQueueEngine connects to the queue database; skips when Postgres is
// unavailable. The integration leg (PG_MODULES="queue/postgres" nix run
// .#integration-pg) runs it against live Postgres.
func newQueueEngine(t *testing.T) metaengine.Engine {
	t.Helper()

	eng, err := postgres.NewEngine(t.Context(), queueDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	return eng
}

// TestQueuePostgresEngine_Conformance: the queue database as a claim
// substrate — the full ADR-0142 DueClaimer + DedupStore contract on the
// same Postgres database the tasks live in.
func TestQueuePostgresEngine_Conformance(t *testing.T) {
	t.Parallel()

	adttest.AssertDueClaimer(t, []adttest.Factory{{Name: "queue-postgres", Create: newQueueEngine}})
	adttest.AssertDedupStore(t, []adttest.Factory{{Name: "queue-postgres", Create: newQueueEngine}})
}

func TestQueuePostgresEngine_DriverRegistry(t *testing.T) {
	t.Parallel()

	factory, err := metaengine.LookupDriver("queue-postgres")
	if err != nil {
		t.Fatalf("LookupDriver: %v", err)
	}

	eng, err := factory(t.Context(), metaengine.DriverConfig{DSN: queueDSN(t)})
	if err != nil {
		// Only connection-class failures skip (no server); everything else
		// (e.g. a transient DDL race under parallel tests) is a real failure.
		if strings.Contains(err.Error(), "refused") || errors.Is(err, os.ErrDeadlineExceeded) {
			t.Skipf("Postgres not available: %v", err)
		}

		t.Fatalf("factory: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	if eng.Profile().Name != "queue-postgres" {
		t.Fatalf("profile name = %s", eng.Profile().Name)
	}

	if !metaengine.SupportsDueClaims(eng) || !metaengine.SupportsDedup(eng) {
		t.Fatal("queue-postgres engine must expose DueClaimer + DedupStore")
	}

	if _, ok := eng.(metaengine.FactSink); !ok {
		t.Fatal("queue-postgres engine must expose FactSink")
	}
}
