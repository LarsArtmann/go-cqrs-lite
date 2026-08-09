package sqliteengine_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestDoctor_RealSQLiteEngine(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("new sqlite engine: %v", err)
	}

	store, err := metaengine.Plan(
		[]metaengine.Engine{eng},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()

	output := store.Doctor(ctx)

	for _, want := range []string{
		"=== Metaengine Doctor ===",
		"--- Health ---",
		"--- Collections ---",
		"--- Persistence ---",
		"find_task",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("Doctor output missing %q, got:\n%s", want, output)
		}
	}

	// SQLite engine should report as healthy.
	if !strings.Contains(output, "all engines healthy") {
		t.Errorf("Doctor should report healthy SQLite engine, got:\n%s", output)
	}
}
