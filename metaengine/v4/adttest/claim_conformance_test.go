package adttest

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
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
func TestClaimConformance_MapHosts( //nolint:tparallel // scenarios are sequential by design
	t *testing.T,
) {
	t.Parallel()

	newMemory := func(t *testing.T) metaengine.Engine {
		return newMapClaimHost(t, metaengine.NewMemoryEngine())
	}

	// Each suite (AssertDueClaimer, AssertDedupStore) closes the engine its
	// factory produced, so Create MUST build a fresh engine every call — the
	// Factory contract. A unique named in-memory database per engine keeps
	// parallel instances (-count>1) from sharing one process-wide memory db.
	newSQLite := func(t *testing.T) metaengine.Engine {
		db, err := sql.Open("sqlite",
			fmt.Sprintf("file:adttest_%d?mode=memory&cache=shared", time.Now().UnixNano()))
		if err != nil {
			t.Fatalf("open sqlite: %v", err)
		}

		db.SetMaxOpenConns(1) // one connection: the named memory db lives while it exists

		t.Cleanup(func() { _ = db.Close() })

		sq, err := sqliteengine.NewSQLiteEngine(db)
		if err != nil {
			t.Fatalf("NewSQLiteEngine: %v", err)
		}

		return newMapClaimHost(t, sq)
	}

	factories := []Factory{
		{Name: "memory-map", Create: newMemory},
		{Name: "sqlite-map", Create: newSQLite},
	}

	AssertDueClaimer(t, factories)
	AssertDedupStore(t, factories)
}

func TestClaimConformance_MemoryEngine( //nolint:tparallel // scenarios are sequential by design
	t *testing.T,
) {
	t.Parallel()

	// The memory engine itself (not the mapClaimHost wrapper) runs the full
	// conformance suites — it is the degraded reference implementation.
	AssertDueClaimer(t, []Factory{
		{
			Name:   "memory",
			Create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
		},
	})
	AssertDedupStore(t, []Factory{
		{
			Name:   "memory",
			Create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
		},
	})
}

// bareEngine is an engine WITHOUT the ADR-0142 capabilities: the probe
// helpers must report false so capability absence is observable.
type bareEngine struct{ metaengine.Engine }

func TestClaimConformance_ProbesReportAbsence(t *testing.T) {
	t.Parallel()

	// Embedding the Engine INTERFACE promotes only the interface's declared
	// methods, not the extra methods of the wrapped memory engine, so
	// bareEngine really lacks the claim and dedup methods.
	eng := bareEngine{metaengine.NewMemoryEngine()}

	// The wrappee has the capabilities, so a false probe below is the
	// wrapper's doing and not the memory engine's.
	if !metaengine.SupportsDueClaims(metaengine.NewMemoryEngine()) {
		t.Fatal("memory engine must satisfy DueClaimer")
	}

	if !metaengine.SupportsDedup(metaengine.NewMemoryEngine()) {
		t.Fatal("memory engine must satisfy DedupStore")
	}

	if metaengine.SupportsDueClaims(eng) {
		t.Fatal("probe reported DueClaimer for an engine that lacks the methods")
	}

	if metaengine.SupportsDedup(eng) {
		t.Fatal("probe reported DedupStore for an engine that lacks the methods")
	}
}
