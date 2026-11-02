package counterkey

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// Clean-room recheck of the tq row-683 report: a counter fold emitting a
// composite-shaped Delta key ("api|pending") was reported to write a stray
// parent-aggregate row ("api|") through Plan+ApplyRecord while a direct
// CounterBackend.CounterIncrement stored the key verbatim. This harness could
// NOT reproduce the truncation at metaengine v4.17.0/sqliteengine v4.5.3 nor
// at the current pins (metaengine v4.19+ / sqliteengine v4.6): both shapes
// land exactly the emitted keys. Kept as the re-run harness if the report
// resurfaces with a fuller recipe.

type ev struct {
	Project string
	From    string
	At      int64
}

type in struct{}

type row struct {
	ID     string
	Status string
}

type listIn struct {
	Status *string
}

var tasksQuery = metaengine.Query[listIn, row]("tasks",
	metaengine.OnRecordTyped("seed", ev{}, func(_ record.Record, e ev) (int64, row) {
		return e.At, row{ID: e.Project, Status: "pending"}
	}),
	metaengine.OnRecordTyped("T", ev{}, func(_ record.Record, e ev, prev row) row {
		prev.Status = "running"
		return prev
	}),
	metaengine.WithColumnarLayout(),
)

var countsQuery = metaengine.Query[in, map[string]int64]("counts",
	metaengine.OnRecordTyped("seed", ev{}, func(_ record.Record, e ev) metaengine.Delta {
		return metaengine.Delta{e.Project + "|pending": 1}
	}),
	metaengine.OnRecordTyped("T", ev{}, func(_ record.Record, e ev) metaengine.Delta {
		return metaengine.Delta{e.Project + "|running": 1, e.Project + "|" + e.From: -1}
	}),
)

func TestCounterKeyCompositeRepro(t *testing.T) {
	ctx := context.Background()
	dsn := "/tmp/counterkey-repro.db"
	if err := os.Remove(dsn); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}

	eng, err := sqliteengine.NewSQLiteEngineFromDSN(dsn)
	must(t, err)
	st, err := metaengine.Plan([]metaengine.Engine{eng}, tasksQuery, countsQuery)
	must(t, err)

	must(t, st.ApplyRecord(ctx, record.Record{Type: "seed"}, ev{Project: "api", At: 1}))
	must(
		t,
		st.ApplyRecord(ctx, record.Record{Type: "T"}, ev{Project: "api", From: "pending", At: 2}),
	)

	cb, ok := eng.(metaengine.CounterBackend)
	if !ok {
		t.Fatal("engine is not a CounterBackend")
	}
	got, err := cb.CounterGet(ctx, "counts")
	must(t, err)
	t.Logf("counters after seed+transition: %v", got)

	dsn2 := "/tmp/counterkey-repro2.db"
	if err := os.Remove(dsn2); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	eng2, err := sqliteengine.NewSQLiteEngineFromDSN(dsn2)
	must(t, err)
	st2, err := metaengine.Plan([]metaengine.Engine{eng2}, countsQuery)
	must(t, err)
	must(
		t,
		st2.ApplyRecord(ctx, record.Record{Type: "T"}, ev{Project: "api", From: "pending", At: 3}),
	)
	cb2, ok := eng2.(metaengine.CounterBackend)
	if !ok {
		t.Fatal("engine is not a CounterBackend")
	}
	got2, err := cb2.CounterGet(ctx, "counts")
	must(t, err)
	t.Logf("from-only delta store: %v", got2)

	dump(t, dsn)
}

func dump(t *testing.T, dsn string) {
	db, err := sql.Open("sqlite", dsn)
	must(t, err)
	rows, err := db.Query(
		`SELECT collection, key, value FROM meta_counter ORDER BY collection, key`,
	)
	must(t, err)
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var c, k string
		var v int64
		must(t, rows.Scan(&c, &k, &v))
		t.Logf("row: %s %q = %d", c, k, v)
	}
	must(t, rows.Err())
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
