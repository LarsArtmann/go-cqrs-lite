package mysql

import (
	"context"
	"os"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestQueueMySQLEngine_Conformance: the queue database as a claim substrate
// — the full ADR-0142 DueClaimer + DedupStore contract on the same MySQL
// database the tasks live in (queue task semantics stay with queue.Store[T]).
// Live-gated like the queue conformance suite (MYSQL_TEST_DSN).
func TestQueueMySQLEngine_Conformance(t *testing.T) {
	dsn := os.Getenv("MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN not set — skipping MySQL engine conformance (server DSN, e.g. root@tcp(127.0.0.1:3306)/?parseTime=true)")
	}

	newEngine := func(t *testing.T) metaengine.Engine {
		t.Helper()

		eng, err := NewEngine(context.Background(), freshDatabase(t, dsn))
		if err != nil {
			t.Fatalf("NewEngine: %v", err)
		}

		t.Cleanup(func() { _ = eng.Close() })

		return eng
	}

	adttest.AssertDueClaimer(t, []adttest.Factory{{Name: "queue-mysql", Create: newEngine}})
	adttest.AssertDedupStore(t, []adttest.Factory{{Name: "queue-mysql", Create: newEngine}})
}

func TestQueueMySQLEngine_DriverRegistry(t *testing.T) {
	factory, err := metaengine.LookupDriver("queue-mysql")
	if err != nil {
		t.Fatalf("LookupDriver: %v", err)
	}

	if _, err := factory(context.Background(), metaengine.DriverConfig{}); err == nil {
		t.Fatal("queue-mysql factory must reject an empty DSN")
	}

	dsn := os.Getenv("MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN not set — skipping live registry leg")
	}

	eng, err := factory(context.Background(), metaengine.DriverConfig{DSN: freshDatabase(t, dsn)})
	if err != nil {
		t.Fatalf("factory: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	if eng.Profile().Name != "queue-mysql" {
		t.Fatalf("profile name = %s", eng.Profile().Name)
	}

	if !metaengine.SupportsDueClaims(eng) || !metaengine.SupportsDedup(eng) {
		t.Fatal("queue-mysql engine must expose DueClaimer + DedupStore")
	}

	if _, ok := eng.(metaengine.FactSink); !ok {
		t.Fatal("queue-mysql engine must expose FactSink")
	}
}
