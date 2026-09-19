package adttest

import (
	"database/sql"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
)

// mapClaimHost is the canonical engine-wiring pattern for map-shaped engines
// (ADR-0142 amendment): embed the Engine plus the shared Map runtimes, and the
// engine satisfies metaengine.DueClaimer + metaengine.DedupStore by method
// promotion — no hand-written claim code.
type mapClaimHost struct {
	metaengine.Engine

	*metaengine.MapDueClaimer

	*metaengine.MapDedupStore
}

func newMapClaimHost(t *testing.T, eng metaengine.Engine) metaengine.Engine {
	t.Helper()

	claimer, err := metaengine.NewMapDueClaimer(eng)
	if err != nil {
		t.Fatalf("NewMapDueClaimer: %v", err)
	}

	dedup, err := metaengine.NewMapDedupStore(eng)
	if err != nil {
		t.Fatalf("NewMapDedupStore: %v", err)
	}

	return mapClaimHost{Engine: eng, MapDueClaimer: claimer, MapDedupStore: dedup}
}

// TestClaimConformance_MapHosts proves the suites themselves against the
// degraded Map runtimes over both stored-value shapes (memory structs, SQLite
// JSON). Engine modules run the same suites over their native engines.
func TestClaimConformance_MapHosts(t *testing.T) {
	t.Parallel()

	newMemory := func(t *testing.T) metaengine.Engine {
		return newMapClaimHost(t, metaengine.NewMemoryEngine())
	}

	factories := []Factory{{Name: "memory-map", Create: newMemory}}

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	db.SetMaxOpenConns(1) // one shared in-memory database across the pool

	t.Cleanup(func() { _ = db.Close() })

	sq, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	t.Cleanup(func() { _ = sq.Close() })

	factories = append(factories, Factory{
		Name:   "sqlite-map",
		Create: func(t *testing.T) metaengine.Engine { return newMapClaimHost(t, sq) },
	})

	AssertDueClaimer(t, factories)
	AssertDedupStore(t, factories)
}

func TestClaimConformance_MissingCapabilityFails(t *testing.T) {
	t.Parallel()

	// A bare engine without the capability must FAIL the suite (asserted
	// capability, not silently skipped) — verified by asserting the suite's
	// behavior on a non-capable engine.
	eng := metaengine.NewMemoryEngine()
	t.Cleanup(func() { _ = eng.Close() })

	if metaengine.SupportsDueClaims(eng) {
		t.Fatal("bare memory engine must not satisfy DueClaimer before wiring")
	}

	if metaengine.SupportsDedup(eng) {
		t.Fatal("bare memory engine must not satisfy DedupStore before wiring")
	}
}
