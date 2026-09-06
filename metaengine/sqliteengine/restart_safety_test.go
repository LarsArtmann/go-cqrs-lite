package sqliteengine_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestSQLiteRestartSafety_StreamAndJournal verifies that reopening a persistent
// SQLite database does NOT reset seq counters to zero — which would cause
// silent key collisions and data loss (see enginetest.RunRestartSafetyTest).
func TestSQLiteRestartSafety_StreamAndJournal(t *testing.T) {
	t.Parallel()

	enginetest.RunRestartSafetyTest(t, func(path string) (metaengine.Engine, error) {
		return sqliteengine.NewSQLiteEngineFromDSN(path)
	})
}

// TestSQLiteRestartSafety_FromDB verifies seq seeding when using
// NewSQLiteEngine (caller-owned *sql.DB path).
func TestSQLiteRestartSafety_FromDB(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "sqlite.db")

	// Phase 1: Open via FromDSN, write, close.
	eng1, err := sqliteengine.NewSQLiteEngineFromDSN(dir)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}

	slb1, ok := eng1.(metaengine.StreamLogBackend)
	if !ok {
		t.Fatal("engine must implement StreamLogBackend")
	}

	if err := slb1.StreamAppend(ctx, "events", "s1", []any{"a", "b"}); err != nil {
		t.Fatalf("first StreamAppend: %v", err)
	}

	if err := eng1.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}

	// Phase 2: Open a raw *sql.DB on the same file, wrap via NewSQLiteEngine,
	// append more.
	db, err := sql.Open("sqlite", dir)
	if err != nil {
		t.Fatalf("raw sqlite open: %v", err)
	}

	eng2, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("FromDB open: %v", err)
	}

	defer func() { _ = eng2.Close() }()

	slb2, ok := eng2.(metaengine.StreamLogBackend)
	if !ok {
		t.Fatal("reopened engine must implement StreamLogBackend")
	}

	if err := slb2.StreamAppend(ctx, "events", "s1", []any{"c"}); err != nil {
		t.Fatalf("post-restart StreamAppend: %v", err)
	}

	ver, err := slb2.StreamVersion(ctx, "events", "s1")
	if err != nil {
		t.Fatalf("StreamVersion after restart: %v", err)
	}

	if ver != 3 {
		t.Fatalf("FromDB restart: stream version = %d, want 3", ver)
	}

	values, err := slb2.StreamRead(ctx, "events", "s1")
	if err != nil {
		t.Fatalf("StreamRead after restart: %v", err)
	}

	if len(values) != 3 {
		t.Fatalf("FromDB restart: stream should retain all 3 events, got %d", len(values))
	}
}
