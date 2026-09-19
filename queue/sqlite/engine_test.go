package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
	sqlite "github.com/larsartmann/go-cqrs-lite/queue/sqlite/v4"
)

func newEngine(t *testing.T) metaengine.Engine {
	t.Helper()

	eng, err := sqlite.NewEngine(filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	return eng
}

// TestQueueSQLiteEngine_Conformance: the queue database as a claim substrate
// — the full ADR-0142 DueClaimer + DedupStore contract on the same SQLite
// file the tasks live in (queue task semantics stay with queue.Store[T]).
func TestQueueSQLiteEngine_Conformance(t *testing.T) {
	t.Parallel()

	adttest.AssertDueClaimer(t, []adttest.Factory{{Name: "queue-sqlite", Create: newEngine}})
	adttest.AssertDedupStore(t, []adttest.Factory{{Name: "queue-sqlite", Create: newEngine}})
}

func TestQueueSQLiteEngine_DriverRegistry(t *testing.T) {
	t.Parallel()

	factory, err := metaengine.LookupDriver("queue-sqlite")
	if err != nil {
		t.Fatalf("LookupDriver: %v", err)
	}

	eng, err := factory(context.Background(), metaengine.DriverConfig{
		DSN: filepath.Join(t.TempDir(), "queue.db"),
	})
	if err != nil {
		t.Fatalf("factory: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	if eng.Profile().Name != "queue-sqlite" {
		t.Fatalf("profile name = %s", eng.Profile().Name)
	}

	if !metaengine.SupportsDueClaims(eng) || !metaengine.SupportsDedup(eng) {
		t.Fatal("queue-sqlite engine must expose DueClaimer + DedupStore")
	}

	if _, ok := eng.(metaengine.FactSink); !ok {
		t.Fatal("queue-sqlite engine must expose FactSink")
	}
}
