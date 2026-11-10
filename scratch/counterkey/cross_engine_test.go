package counterkey

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/mysqlengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// Cross-engine sweep closing the search space the sqlite-only recheck left
// open: the 04-18 "truncate to first component" report never named its
// engine, so the composite-key harness shape runs against every backend that
// can hold a counter store. Expected outcome for each backend: the emitted
// Delta keys land verbatim ("api|pending", "api|running"); no truncated
// parent row ("api|"). Engines without a CounterBackend skip BEFORE any fold
// runs: a counter query on a non-CounterBackend engine errors by design, so
// the capability assertion must precede Plan/ApplyRecord (turso participates
// — it delegates storage to the sqlite engine, counter backend included).
// CGO_ENABLED=1 is required for the duckdb binding; the others build pure.
// The pg and mysql legs read the canonical engine-suite env rails
// (POSTGRES_TEST_DSN, MYSQL_TEST_DSN — e.g.
// "root:pass@tcp(localhost:3306)/test?parseTime=true") and skip when unset.

type engineFactory struct {
	name string
	open func(t *testing.T) metaengine.Engine
}

func engineFactories(t *testing.T) []engineFactory {
	t.Helper()
	factories := []engineFactory{
		{"memory", func(t *testing.T) metaengine.Engine {
			return metaengine.NewMemoryEngine()
		}},
		{"pebble", func(t *testing.T) metaengine.Engine {
			dir := t.TempDir()
			eng, err := pebbleengine.NewPebbleEngine(filepath.Join(dir, "db"))
			must(t, err)

			return eng
		}},
		{"badger", func(t *testing.T) metaengine.Engine {
			dir := t.TempDir()
			eng, err := badgerengine.NewBadgerEngine(filepath.Join(dir, "db"))
			must(t, err)

			return eng
		}},
		{"duckdb", func(t *testing.T) metaengine.Engine {
			dsn := filepath.Join(t.TempDir(), "db.duckdb")
			eng, err := duckdbengine.New(dsn)
			must(t, err)

			return eng
		}},
		{"bbolt", func(t *testing.T) metaengine.Engine {
			dir := t.TempDir()
			eng, err := bboltengine.NewBboltEngine(filepath.Join(dir, "db"))
			must(t, err)

			return eng
		}},
		{"turso", func(t *testing.T) metaengine.Engine {
			eng, err := tursoengine.New(filepath.Join(t.TempDir(), "db"))
			must(t, err)

			return eng
		}},
		{"pg", func(t *testing.T) metaengine.Engine {
			dsn := os.Getenv("POSTGRES_TEST_DSN")
			if dsn == "" {
				t.Skip("POSTGRES_TEST_DSN not set — skipping pg counter-key leg")
			}
			eng, err := pgengine.New(dsn)
			must(t, err)

			return eng
		}},
		{"mysql", func(t *testing.T) metaengine.Engine {
			dsn := os.Getenv("MYSQL_TEST_DSN")
			if dsn == "" {
				t.Skip("MYSQL_TEST_DSN not set — skipping mysql counter-key leg")
			}
			eng, err := mysqlengine.New(dsn)
			must(t, err)

			return eng
		}},
	}

	return factories
}

func TestCounterKeyCompositeCrossEngine(t *testing.T) {
	ctx := context.Background()
	for _, f := range engineFactories(t) {
		t.Run(f.name, func(t *testing.T) {
			eng := f.open(t)

			cb, ok := eng.(metaengine.CounterBackend)
			if !ok {
				t.Skipf("engine %s does not implement CounterBackend — no counter store to inspect, class structurally out of reach", f.name)
			}

			st, err := metaengine.Plan([]metaengine.Engine{eng}, tasksQuery, countsQuery)
			must(t, err)
			must(t, st.ApplyRecord(ctx, record.Record{Type: "seed"}, ev{Project: "api", At: 1}))
			must(
				t,
				st.ApplyRecord(
					ctx,
					record.Record{Type: "T"},
					ev{Project: "api", From: "pending", At: 2},
				),
			)

			got, err := cb.CounterGet(ctx, "counts")
			must(t, err)
			t.Logf("counters after seed+transition: %v", got)

			for _, key := range []string{"api|", "api|pending", "api|running"} {
				want := int64(0)
				switch key {
				case "api|pending":
					want = 0
				case "api|running":
					want = 1
				}
				if key == "api|" && got[key] != 0 {
					t.Fatalf(
						"truncated parent row present: %q = %d (counters: %v)",
						key,
						got[key],
						got,
					)
				}
				if key != "api|" && got[key] != want {
					t.Fatalf(
						"composite key %q = %d, want %d (counters: %v)",
						key,
						got[key],
						want,
						got,
					)
				}
			}

			fromOnly, err := metaengine.Plan([]metaengine.Engine{eng}, countsQuery)
			must(t, err)
			must(
				t,
				fromOnly.ApplyRecord(
					ctx,
					record.Record{Type: "T"},
					ev{Project: "api", From: "pending", At: 3},
				),
			)
			got2, err := cb.CounterGet(ctx, "counts")
			must(t, err)
			t.Logf("from-only delta store: %v", got2)
			if got2["api|"] != 0 {
				t.Fatalf("truncated parent row present after from-only delta: %v", got2)
			}
			if got2["api|pending"] != -1 {
				t.Fatalf(
					"from-side composite key %q = %d, want -1 (counters: %v)",
					"api|pending",
					got2["api|pending"],
					got2,
				)
			}
		})
	}
}
