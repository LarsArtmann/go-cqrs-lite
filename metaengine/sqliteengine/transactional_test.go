package sqliteengine_test

import (
	"database/sql"
	"testing"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
	_ "modernc.org/sqlite"
)

func TestSQLiteEngine_Transactional(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}

	db.SetMaxOpenConns(1)
	defer db.Close()

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	defer func() { _ = eng.Close() }()

	enginetest.RunTransactionalTest(t, eng)
}

func TestSQLiteEngine_ConcurrentTx(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}

	db.SetMaxOpenConns(1)
	defer db.Close()

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	defer func() { _ = eng.Close() }()

	enginetest.RunConcurrentTxTest(t, eng)
}
