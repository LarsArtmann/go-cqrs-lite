package sqliteengine_test

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// newEngine builds a fresh, isolated SQLite engine over a unique named
// in-memory database (parallel instances must not share one process-wide
// memory db, and cached statements must not outlive their database).
func newEngine(t *testing.T) metaengine.Engine {
	t.Helper()

	db, err := sql.Open("sqlite",
		fmt.Sprintf("file:sqliteengine_claims_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	db.SetMaxOpenConns(1)

	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	return eng
}

// TestSQLiteDueClaims pins the engine's ADR-0142 capabilities to the shared
// conformance contract: claimkit provides the implementation, this proves the
// wiring (embedding, capability assertions, profile entries) holds.
func TestSQLiteDueClaims(t *testing.T) {
	t.Parallel()

	adttest.AssertDueClaimer(t, []adttest.Factory{{Name: "sqlite", Create: newEngine}})
	adttest.AssertDedupStore(t, []adttest.Factory{{Name: "sqlite", Create: newEngine}})
}

func TestSQLiteDueClaims_CapabilitySurface(t *testing.T) {
	t.Parallel()

	eng := newEngine(t)

	if !metaengine.SupportsDueClaims(eng) {
		t.Fatal("sqlite engine must satisfy DueClaimer after claimkit wiring")
	}

	if !metaengine.SupportsDedup(eng) {
		t.Fatal("sqlite engine must satisfy DedupStore after claimkit wiring")
	}

	if _, ok := eng.(metaengine.FactSink); !ok {
		t.Fatal("sqlite engine must satisfy FactSink after claimkit wiring")
	}

	complexity, ok := eng.Profile().SupportsADT(metaengine.ADTDueClaim)
	if !ok || complexity != metaengine.ComplexityOLogN {
		t.Fatalf("profile must declare ADTDueClaim at O(logN), got %s/%v", complexity, ok)
	}

	if complexity, ok := eng.Profile().
		SupportsADT(metaengine.ADTDedup); !ok ||
		complexity != metaengine.ComplexityOLogN {
		t.Fatalf("profile must declare ADTDedup at O(logN), got %s/%v", complexity, ok)
	}

	if eng.Profile().IsDegraded(metaengine.ADTDueClaim) {
		t.Fatal("sqlite claims are native (indexed SQL), not degraded")
	}
}
