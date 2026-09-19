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
// StreamLogBackend is required. The Map and Multimap legs run when the engine
// implements those backends and are skipped (with a log note) otherwise.
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

		restartSeedStreamMapMultimapVector(t, ctx, eng1)

		ver1, err := eng1.(metaengine.StreamLogBackend).StreamVersion(ctx, "events", "s1")
		if err != nil {
			t.Fatalf("StreamVersion before close: %v", err)
		}

		if ver1 != 3 {
			t.Fatalf("stream version before close = %d, want 3", ver1)
		}

		journal1, err := eng1.(metaengine.StreamLogBackend).JournalReadAll(ctx, "events")
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

		defer metaengine.DeferClose(eng2)

		restartVerifyAcrossReopen(t, ctx, eng2)
	})
}

// restartSeedStreamMapMultimapVector writes the phase-1 seed data: 3 stream
// events, one map entry, one multimap entry, two metadata-carrying embeddings.
func restartSeedStreamMapMultimapVector(
	t *testing.T,
	ctx context.Context,
	eng metaengine.Engine,
) {
	//art-dupl:accept seed/verify helpers intentionally share the StreamLog capability-guard shape
	t.Helper()

	slb, ok := eng.(metaengine.StreamLogBackend)
	if !ok {
		t.Fatal("engine must implement StreamLogBackend")
	}

	// Append 3 events to stream "s1".
	if err := slb.StreamAppend(ctx, "events", "s1", []any{"e1", "e2", "e3"}); err != nil {
		t.Fatalf("first StreamAppend: %v", err)
	}

	// Map ADT — verify journalSeq seeding doesn't collide.
	if mb, hasMap := eng.(metaengine.MapBackend); hasMap {
		if err := mb.MapSet(ctx, "kv", "key1", "val1"); err != nil {
			t.Fatalf("MapSet: %v", err)
		}
	}

	// Multimap ADT — verify mmSeq seeding doesn't collide.
	if mmb, hasMultimap := eng.(metaengine.MultimapBackend); hasMultimap {
		if err := mmb.MultiAdd(ctx, "mm1", "entry1", "val1"); err != nil {
			t.Fatalf("MultiAdd: %v", err)
		}
	}

	// Vector ADT — verify embeddings (payload + metadata) persist across
	// reopen, not just seq counters.
	if vb, hasVector := eng.(metaengine.VectorBackend); hasVector {
		for _, emb := range []metaengine.Embedding{
			{ID: "v1", Values: []float32{1, 0, 0}, Metadata: map[string]any{"tenant": "a"}},
			{ID: "v2", Values: []float32{0, 1, 0}, Metadata: map[string]any{"tenant": "b"}},
		} {
			if err := vb.VectorInsert(ctx, "vecs", emb); err != nil {
				t.Fatalf("first VectorInsert %s: %v", emb.ID, err)
			}
		}
	}
}

// restartVerifyAcrossReopen runs the phase-2 assertions on the reopened
// engine: 2 more stream events (seq-seeding proof), all 5 entries retained,
// then Map/Multimap/Vector persistence and the post-restart dimension lock.
func restartVerifyAcrossReopen(
	t *testing.T,
	ctx context.Context,
	eng metaengine.Engine,
) {
	t.Helper()

	slb2, ok := eng.(metaengine.StreamLogBackend)
	if !ok {
		t.Fatal("reopened engine must implement StreamLogBackend")
	}

	// Append 2 MORE events — without seq seeding these would overwrite seqs 1-2.
	if err := slb2.StreamAppend(ctx, "events", "s1", []any{"e4", "e5"}); err != nil {
		t.Fatalf("post-restart StreamAppend: %v", err)
	}

	values, err := slb2.StreamRead(ctx, "events", "s1")
	if err != nil {
		t.Fatalf("StreamRead after restart: %v", err)
	}

	if len(values) != 5 {
		t.Fatalf("stream should retain all 5 events after restart, got %d", len(values))
	}

	ver2, err := slb2.StreamVersion(ctx, "events", "s1")
	if err != nil {
		t.Fatalf("StreamVersion after restart: %v", err)
	}

	if ver2 != 5 {
		t.Fatalf("stream version after restart = %d, want 5", ver2)
	}

	journal2, err := slb2.JournalReadAll(ctx, "events")
	if err != nil {
		t.Fatalf("JournalReadAll after restart: %v", err)
	}

	if len(journal2) != 5 {
		t.Fatalf("journal should retain all 5 entries after restart, got %d", len(journal2))
	}

	restartVerifyMapMultimap(t, ctx, eng)
	restartVerifyVector(t, ctx, eng)
}

// restartVerifyMapMultimap pins Map/Multimap persistence across reopen.
func restartVerifyMapMultimap(
	t *testing.T,
	ctx context.Context,
	eng metaengine.Engine,
) {
	t.Helper()

	// Verify Map ADT data survived.
	if mb2, hasMap := eng.(metaengine.MapBackend); hasMap {
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
	}

	// Verify new Multimap entry doesn't collide with existing.
	if mmb2, hasMultimap := eng.(metaengine.MultimapBackend); hasMultimap {
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
	}
}

// restartVerifyVector pins Vector persistence across reopen: k-NN ordering
// survives, and post-restart inserts pass the dimension lock re-read from
// persisted rows.
func restartVerifyVector(
	t *testing.T,
	ctx context.Context,
	eng metaengine.Engine,
) {
	t.Helper()

	vb2, hasVector := eng.(metaengine.VectorBackend)
	if !hasVector {
		return
	}

	results, err := vb2.VectorSearch(ctx, "vecs", []float32{1, 0, 0}, 2, "cosine")
	if err != nil {
		t.Fatalf("VectorSearch after restart: %v", err)
	}

	if len(results) != 2 || results[0].ID != "v1" || results[1].ID != "v2" {
		t.Fatalf("vectors should survive restart with ordering, got %+v", results)
	}

	if err := vb2.VectorInsert(
		ctx,
		"vecs",
		metaengine.Embedding{ID: "v3", Values: []float32{0, 0, 1}},
	); err != nil {
		t.Fatalf("post-restart VectorInsert: %v", err)
	}
}

// RunRestartSafetyFromDBTest verifies seq seeding across a CALLER-OWNED-DB
// reopen — the FromDB constructor path whose seeding behavior can diverge
// from the DSN/path constructor. The scenario:
//
//  1. open(dir) constructs the engine (module-specific filename inside dir)
//  2. Two stream values are appended, then the engine is closed
//  3. reopenRaw(dir) reopens the RAW DB handle on the same file and wraps it
//     via the engine's FromDB constructor
//  4. A third value is appended; version must be 3 (not 1) and all three
//     values must survive
//
// StreamLogBackend is required on both paths. Engines that need an
// availability probe (e.g. CGo DuckDB) keep it in the caller before invoking
// this harness.
func RunRestartSafetyFromDBTest(
	t *testing.T,
	open, reopenRaw func(dir string) (metaengine.Engine, error),
) {
	t.Helper()

	ctx := context.Background()
	dir := t.TempDir()

	// --- Phase 1: constructor path, write, close ---
	eng1, err := open(dir)
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

	// --- Phase 2: caller-owned-DB reopen, append, verify ---
	eng2, err := reopenRaw(dir)
	if err != nil {
		t.Fatalf("FromDB reopen: %v", err)
	}

	defer metaengine.DeferClose(eng2)

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
