package bench_test

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite" // register sqlite driver for all metaengine tests

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func newSQLiteEngine() (metaengine.Engine, *sql.DB) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(1)
	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		panic(err)
	}
	return eng, db
}

func newSQLiteEngineForPath(path string) (metaengine.Engine, *sql.DB) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(1)
	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		panic(err)
	}
	return eng, db
}

func newPlannedSQLiteEngine(
	t *testing.T,
	plans []metaengine.LayoutPlan,
) (metaengine.Engine, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	eng, err := sqliteengine.NewPlannedSQLiteEngine(db, plans)
	if err != nil {
		t.Fatalf("NewPlannedSQLiteEngine: %v", err)
	}
	return eng, db
}
