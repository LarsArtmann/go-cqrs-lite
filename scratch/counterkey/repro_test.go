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

type ev struct {
	Project string
	From    string
	At      int64
}

type in struct{}

var sep = "|"

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
		return metaengine.Delta{e.Project + sep + "pending": 1}
	}),
	metaengine.OnRecordTyped("T", ev{}, func(_ record.Record, e ev) metaengine.Delta {
		return metaengine.Delta{e.Project + sep + "running": 1, e.Project + sep + e.From: -1}
	}),
)

func TestCounterKeyCompositeRepro(t *testing.T) {
	for _, sep := range []string{"|", "_", "\x1f"} {
		t.Logf("=== separator %q ===\n", sep)
		run(sep)
	}
}

func run(sep string) {
	ctx := context.Background()
	dsn := "/tmp/counterkey-repro.db"
	_ = os.Remove(dsn)

	eng, err := sqliteengine.NewSQLiteEngineFromDSN(dsn)
	must(t, err)
	st, err := metaengine.Plan([]metaengine.Engine{eng}, tasksQuery, countsQuery)
	must(t, err)

	_ = st
	_ = sep

	must(st.ApplyRecord(ctx, record.Record{Type: "seed"}, ev{Project: "api"}))
	must(st.ApplyRecord(ctx, record.Record{Type: "T"}, ev{Project: "api", From: "pending"}))

	cb, ok := eng.(metaengine.CounterBackend)
	must2(t, ok)
	got, err := cb.CounterGet(ctx, "counts")
	must(t, err)
	t.Logf("counters after seed+transition: %v", got)

	// literal repro shape: from-only single-key delta through a fresh store
	dsn2 := "/tmp/counterkey-repro2.db"
	_ = os.Remove(dsn2)
	eng2, err := sqliteengine.NewSQLiteEngineFromDSN(dsn2)
	must(t, err)
	st2, err := metaengine.Plan([]metaengine.Engine{eng2}, countsQuery)
	must(t, err)
	must(st2.ApplyRecord(ctx, record.Record{Type: "T"}, ev{Project: "api", From: "pending"}))
	cb2, _ := eng2.(metaengine.CounterBackend)
	got2, err := cb2.CounterGet(ctx, "counts")
	must(t, err)
	t.Logf("from-only delta store: %v", got2)

	dump(t, dsn)
}

func dump(t interface{ Logf(string, ...interface{}) }, dsn string) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Logf("open dump db: %v", err)
		return
	}
	rows, err := db.Query(
		`SELECT collection, key, value FROM meta_counter ORDER BY collection, key`,
	)
	must(t, err)
	for rows.Next() {
		var c, k string
		var v int64
		if err := rows.Scan(&c, &k, &v); err != nil {
		t.Logf("scan: %v", err)
		return
	}
		t.Logf("row: %s %q = %d\n", c, k, v)
	}
	must(t, rows.Err())
}

func must(t *testing.T, err error) {
	if err != nil {
		panic(err)
	}
}

func must2(t *testing.T, b bool) {
	if !b {
		panic("not a CounterBackend")
	}
}
