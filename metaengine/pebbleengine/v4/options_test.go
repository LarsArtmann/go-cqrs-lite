package pebbleengine

import (
	"context"
	"testing"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
)

func TestNewPebbleEngine_DefaultSyncWrites(t *testing.T) {
	t.Parallel()

	eng, err := NewPebbleEngine("")
	if err != nil {
		t.Fatalf("NewPebbleEngine failed: %v", err)
	}

	defer func() { _ = eng.Close() }()

	pe := eng.(*pebbleEngine)
	if !pe.syncWrites {
		t.Fatal("engine should default to sync writes")
	}

	if pe.writeOptions() == nil {
		t.Fatal("writeOptions() should be pebble.Sync by default")
	}
}

func TestNewPebbleEngine_WithAsyncWrites(t *testing.T) {
	t.Parallel()

	eng, err := NewPebbleEngine("", WithAsyncWrites())
	if err != nil {
		t.Fatalf("NewPebbleEngine failed: %v", err)
	}

	defer func() { _ = eng.Close() }()

	pe := eng.(*pebbleEngine)
	if pe.syncWrites {
		t.Fatal("WithAsyncWrites should disable sync writes")
	}

	if pe.writeOptions() != nil {
		t.Fatal("writeOptions() should be nil with WithAsyncWrites")
	}

	if err := pe.MapSet(context.Background(), "col", "k", map[string]any{"v": 1}); err != nil {
		t.Fatalf("MapSet with async writes failed: %v", err)
	}

	val, ok, err := pe.MapGet(context.Background(), "col", "k")
	if err != nil || !ok {
		t.Fatalf("MapGet after async MapSet: ok=%v err=%v", ok, err)
	}

	decoded, _ := val.(map[string]any)
	if decoded["v"] != float64(1) {
		t.Fatalf("roundtrip mismatch: got %v", val)
	}
}

func TestNewPebbleEngineFromDB_WithAsyncWrites(t *testing.T) {
	t.Parallel()

	db, err := pebble.Open("", &pebble.Options{FS: vfs.NewMem()})
	if err != nil {
		t.Fatalf("open in-memory pebble failed: %v", err)
	}

	defer func() { _ = db.Close() }()

	eng, err := NewPebbleEngineFromDB(db, WithAsyncWrites())
	if err != nil {
		t.Fatalf("NewPebbleEngineFromDB failed: %v", err)
	}

	if eng.(*pebbleEngine).syncWrites {
		t.Fatal("WithAsyncWrites should apply when wrapping an existing DB")
	}
}
