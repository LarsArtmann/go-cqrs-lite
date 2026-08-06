package metaengine_test

import (
	"database/sql"

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
