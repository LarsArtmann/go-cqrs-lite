package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

type ev struct{ Project string }

type in struct{}

var q = metaengine.Query[in, map[string]int64]("counts",
	metaengine.OnRecordTyped("T", ev{}, func(_ record.Record, e ev) metaengine.Delta {
		return metaengine.Delta{"api|pending": -1}
	}),
)

func main() {
	ctx := context.Background()
	dsn := "/tmp/counterkey-repro.db"
	_ = os.Remove(dsn)

	eng, err := sqliteengine.NewSQLiteEngineFromDSN(dsn)
	must(err)
	st, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	must(err)

	must(st.ApplyRecord(ctx, record.Record{Type: "T"}, ev{Project: "api"}))

	cb, ok := eng.(metaengine.CounterBackend)
	must2(ok)
	must(cb.CounterIncrement(ctx, "counts", metaengine.Delta{"api|pending": -1}))

	got, err := cb.CounterGet(ctx, "counts")
	must(err)
	fmt.Println("after ApplyRecord + direct increment:", got)

	db, err := sql.Open("sqlite", dsn)
	must(err)
	rows, err := db.Query(`SELECT key, value FROM meta_counter WHERE collection='counts'`)
	must(err)
	for rows.Next() {
		var k string
		var v int64
		must(rows.Scan(&k, &v))
		fmt.Printf("row: %q = %d\n", k, v)
	}
	must(rows.Err())
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func must2(b bool) {
	if !b {
		panic("not a CounterBackend")
	}
}
