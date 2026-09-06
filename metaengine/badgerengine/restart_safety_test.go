package badgerengine_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/dgraph-io/badger/v4"

	"github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestBadgerRestartSafety_StreamAndJournal verifies that reopening a persistent
// Badger DB does NOT reset seq counters to zero — which would cause silent key
// collisions and data loss (see enginetest.RunRestartSafetyTest).
func TestBadgerRestartSafety_StreamAndJournal(t *testing.T) {
	t.Parallel()

	enginetest.RunRestartSafetyTest(t, func(path string) (metaengine.Engine, error) {
		return badgerengine.NewBadgerEngine(path)
	})
}

// TestBadgerRestartSafety_FromDB verifies seq seeding when using
// NewBadgerEngineFromDB (caller-owned DB path).
func TestBadgerRestartSafety_FromDB(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "badger")

	// Phase 1: Open via NewBadgerEngine, write, close.
	eng1, err := badgerengine.NewBadgerEngine(dir)
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

	// Phase 2: Open raw DB, wrap via NewBadgerEngineFromDB, append more.
	db, err := badger.Open(badger.DefaultOptions(dir).WithLogger(nil))
	if err != nil {
		t.Fatalf("raw badger open: %v", err)
	}

	eng2, err := badgerengine.NewBadgerEngineFromDB(db)
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
