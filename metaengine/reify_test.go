package metaengine_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// recordResult has an unexported field that is lost when the value crosses
// the SQLite JSON boundary. This test documents that caveat (ADR-0066).
type recordResult struct {
	ID     string
	secret string // unexported — lost on JSON round-trip through SQLite
}

type recordEvent struct {
	ID string
}

type recordQuery struct {
	ID string
}

func newSQLiteEngineForStd(t *testing.T) (metaengine.Engine, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	eng, err := metaengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("new sqlite engine: %v", err)
	}
	return eng, db
}

func TestExecuteTyped_SQLite_UnexportedFieldsLost(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	eng, db := newSQLiteEngineForStd(t)
	defer db.Close()

	query := metaengine.Query[recordQuery, recordResult](
		"record_lookup",
		metaengine.On(recordEvent{}, func(e recordEvent) (string, recordResult) {
			return e.ID, recordResult{ID: e.ID, secret: "hidden-value"}
		}),
	)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, query)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	store.Apply(ctx, recordEvent{ID: "rec-1"})

	result, err := metaengine.ExecuteTyped[recordResult](
		ctx, store, "record_lookup", recordQuery{ID: "rec-1"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}
	if result.ID != "rec-1" {
		t.Errorf("ID: got %q, want %q", result.ID, "rec-1")
	}
	// The unexported 'secret' field is lost after JSON round-trip through SQLite.
	// This documents the caveat in ADR-0066: only exported fields survive.
	if result.secret != "" {
		t.Errorf("unexported field survived JSON round-trip: got %q, expected empty", result.secret)
	}
}

func BenchmarkExecuteTyped_SQLite_Reify(b *testing.B) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		b.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	eng, err := metaengine.NewSQLiteEngine(db)
	if err != nil {
		b.Fatalf("new sqlite engine: %v", err)
	}

	query := metaengine.Query[recordQuery, recordResult](
		"bench_reify",
		metaengine.On(recordEvent{}, func(e recordEvent) (string, recordResult) {
			return e.ID, recordResult{ID: e.ID, secret: "hidden"}
		}),
	)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, query)
	if err != nil {
		b.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	store.Apply(ctx, recordEvent{ID: "bench-1"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := metaengine.ExecuteTyped[recordResult](
			ctx, store, "bench_reify", recordQuery{ID: "bench-1"},
		)
		if err != nil {
			b.Fatalf("ExecuteTyped: %v", err)
		}
	}
}
