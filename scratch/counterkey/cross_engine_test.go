package counterkey

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// Cross-engine sweep closing the search space the sqlite-only recheck left
// open: the 04-18 "truncate to first component" report never named its engine,
// so the composite-key harness shape runs against every pure-Go backend plus
// duckdb. Expected outcome for each backend: the emitted Delta keys land
// verbatim ("api|pending", "api|running"); no truncated parent row ("api|").
// CGO_ENABLED=1 is required for the duckdb binding; the others build pure.

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
	}
	return factories
}

func TestCounterKeyCompositeCrossEngine(t *testing.T) {
	ctx := context.Background()
	for _, f := range engineFactories(t) {
		f := f
		t.Run(f.name, func(t *testing.T) {
			eng := f.open(t)

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

			cb, ok := eng.(metaengine.CounterBackend)
			if !ok {
				t.Skipf("engine %s does not implement CounterBackend", f.name)
			}
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
