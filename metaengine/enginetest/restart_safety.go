package enginetest

import (
	"context"
	"path/filepath"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// RestartSafetyFactory opens a persistent engine rooted at path. The
// restart-safety suite relies on every call against the same path resuming
// the persisted seq counters instead of resetting them.
type RestartSafetyFactory func(path string) (metaengine.Engine, error)

// RunRestartSafetyTest verifies that reopening a persistent engine does NOT
// reset seq counters to zero — which would cause silent key collisions and
// data loss. The scenario:
//
//  1. Opens a persistent engine on disk
//  2. Appends events to stream "s1" (version 3) + writes Map + Multimap entries
//  3. Closes the engine
//  4. Reopens on the same path
//  5. Appends MORE events to "s1" and the same Map/Multimap collections
//  6. Verifies no data was overwritten — stream has 5 events, journal has 5 entries
//
// The caller is responsible for closing engines returned by the factory that
// the harness does not close itself.
func RunRestartSafetyTest(t *testing.T, newEngine RestartSafetyFactory) {
	t.Helper()

	t.Run("StreamAndJournal", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		dir := filepath.Join(t.TempDir(), "engine")

		// --- Phase 1: Write data, close ---
		eng1, err := newEngine(dir)
		if err != nil {
			t.Fatalf("first open: %v", err)
		}

		slb1, ok := eng1.(metaengine.StreamLogBackend)
		if !ok {
			t.Fatal("engine must implement StreamLogBackend")
		}

		mb1, ok := eng1.(metaengine.MapBackend)
		if !ok {
			t.Fatal("engine must implement MapBackend")
		}

		mmb1, ok := eng1.(metaengine.MultimapBackend)
		if !ok {
			t.Fatal("engine must implement MultimapBackend")
		}

		// Append 3 events to stream "s1".
		if err := slb1.StreamAppend(ctx, "events", "s1", []any{"e1", "e2", "e3"}); err != nil {
			t.Fatalf("first StreamAppend: %v", err)
		}

		// Map ADT — verify journalSeq seeding doesn't collide.
		if err := mb1.MapSet(ctx, "kv", "key1", "val1"); err != nil {
			t.Fatalf("MapSet: %v", err)
		}

		// Multimap ADT — verify mmSeq seeding doesn't collide.
		if err := mmb1.MultiAdd(ctx, "mm1", "entry1", "val1"); err != nil {
			t.Fatalf("MultiAdd: %v", err)
		}

		ver1, err := slb1.StreamVersion(ctx, "events", "s1")
		if err != nil {
			t.Fatalf("StreamVersion before close: %v", err)
		}

		if ver1 != 3 {
			t.Fatalf("stream version before close = %d, want 3", ver1)
		}

		journal1, err := slb1.JournalReadAll(ctx, "events")
		if err != nil {
			t.Fatalf("JournalReadAll before close: %v", err)
		}

		if len(journal1) != 3 {
			t.Fatalf("journal entries before close = %d, want 3", len(journal1))
		}

		if err := eng1.Close(); err != nil {
			t.Fatalf("close first engine: %v", err)
		}

		// --- Phase 2: Reopen and append more ---
		eng2, err := newEngine(dir)
		if err != nil {
			t.Fatalf("reopen: %v", err)
		}

		defer func() { _ = eng2.Close() }()

		slb2, ok := eng2.(metaengine.StreamLogBackend)
		if !ok {
			t.Fatal("reopened engine must implement StreamLogBackend")
		}

		mb2, ok := eng2.(metaengine.MapBackend)
		if !ok {
			t.Fatal("reopened engine must implement MapBackend")
		}

		mmb2, ok := eng2.(metaengine.MultimapBackend)
		if !ok {
			t.Fatal("reopened engine must implement MultimapBackend")
		}

		// Append 2 MORE events — without seq seeding these would overwrite seqs 1-2.
		if err := slb2.StreamAppend(ctx, "events", "s1", []any{"e4", "e5"}); err != nil {
			t.Fatalf("post-restart StreamAppend: %v", err)
		}

		// Verify stream has ALL 5 events (not 2 — which would mean overwrites).
		values, err := slb2.StreamRead(ctx, "events", "s1")
		if err != nil {
			t.Fatalf("StreamRead after restart: %v", err)
		}

		if len(values) != 5 {
			t.Fatalf("stream should retain all 5 events after restart, got %d", len(values))
		}

		// Verify version is 5 (not 2).
		ver2, err := slb2.StreamVersion(ctx, "events", "s1")
		if err != nil {
			t.Fatalf("StreamVersion after restart: %v", err)
		}

		if ver2 != 5 {
			t.Fatalf("stream version after restart = %d, want 5", ver2)
		}

		// Verify journal has ALL 5 entries in order.
		journal2, err := slb2.JournalReadAll(ctx, "events")
		if err != nil {
			t.Fatalf("JournalReadAll after restart: %v", err)
		}

		if len(journal2) != 5 {
			t.Fatalf("journal should retain all 5 entries after restart, got %d", len(journal2))
		}

		// Verify Map ADT data survived.
		mapVal, found, err := mb2.MapGet(ctx, "kv", "key1")
		if err != nil {
			t.Fatalf("MapGet after restart: %v", err)
		}

		if !found {
			t.Fatal("Map key1 should exist after restart")
		}

		if mapVal != "val1" {
			t.Fatalf("Map data should survive restart, got %v", mapVal)
		}

		// Verify new Map write doesn't overwrite existing.
		if err := mb2.MapSet(ctx, "kv", "key2", "val2"); err != nil {
			t.Fatalf("MapSet key2: %v", err)
		}

		mapVal2, found2, err := mb2.MapGet(ctx, "kv", "key2")
		if err != nil {
			t.Fatalf("MapGet key2: %v", err)
		}

		if !found2 {
			t.Fatal("Map key2 should exist after write")
		}

		if mapVal2 != "val2" {
			t.Fatalf("Map key2 = %v, want val2", mapVal2)
		}

		// Verify new Multimap entry doesn't collide with existing.
		if err := mmb2.MultiAdd(ctx, "mm1", "entry1", "val2"); err != nil {
			t.Fatalf("MultiAdd after restart: %v", err)
		}

		mmVals, err := mmb2.MultiGet(ctx, "mm1", "entry1")
		if err != nil {
			t.Fatalf("MultiGet after restart: %v", err)
		}

		if len(mmVals) != 2 {
			t.Fatalf("multimap should have 2 values after restart append, got %d", len(mmVals))
		}
	})
}
