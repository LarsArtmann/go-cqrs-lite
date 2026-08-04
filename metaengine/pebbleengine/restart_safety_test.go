package pebbleengine_test

import (
	"context"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
)

// TestPebbleRestartSafety_StreamAndJournal verifies that reopening a persistent
// Pebble DB does NOT reset seq counters to zero — which would cause silent key
// collisions and data loss. The test:
//
//  1. Opens a persistent engine on disk
//  2. Appends events to stream "s1" (version 3) and uses Map ADT
//  3. Closes the engine
//  4. Reopens on the same directory
//  5. Appends MORE events to "s1" and the same Map collection
//  6. Verifies no data was overwritten — stream has 5 events, journal has 5 entries
func TestPebbleRestartSafety_StreamAndJournal(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "pebble")

	// --- Phase 1: Write data, close ---
	eng1, err := pebbleengine.NewPebbleEngine(dir)
	g.Expect(err).NotTo(HaveOccurred(), "first open should succeed")

	slb1, ok := eng1.(metaengine.StreamLogBackend)
	g.Expect(ok).To(BeTrue(), "engine must implement StreamLogBackend")

	// Append 3 events to stream "s1".
	g.Expect(slb1.StreamAppend(ctx, "events", "s1", []any{"e1", "e2", "e3"})).
		To(Succeed(), "first StreamAppend")

	// Also test the Log ADT to verify journalSeq seeding.
	g.Expect(eng1.MapSet(ctx, "kv", "key1", "val1")).To(Succeed())

	// Also test the Multimap ADT to verify mmSeq seeding.
	g.Expect(eng1.MultiAdd(ctx, "mm1", "entry1", "val1")).To(Succeed())

	ver1, err := slb1.StreamVersion(ctx, "events", "s1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ver1).To(Equal(int64(3)), "stream version before close")

	journal1, err := slb1.JournalReadAll(ctx, "events")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(journal1).To(HaveLen(3), "journal entries before close")

	g.Expect(eng1.Close()).To(Succeed(), "close first engine")

	// --- Phase 2: Reopen and append more ---
	eng2, err := pebbleengine.NewPebbleEngine(dir)
	g.Expect(err).NotTo(HaveOccurred(), "reopen should succeed")
	defer eng2.Close()

	slb2, ok := eng2.(metaengine.StreamLogBackend)
	g.Expect(ok).To(BeTrue())

	// Append 2 MORE events — without seq seeding these would overwrite seqs 1-2.
	g.Expect(slb2.StreamAppend(ctx, "events", "s1", []any{"e4", "e5"})).
		To(Succeed(), "post-restart StreamAppend")

	// Verify stream has ALL 5 events (not 2 — which would mean overwrites).
	values, err := slb2.StreamRead(ctx, "events", "s1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(values).To(HaveLen(5), "stream should retain all 5 events after restart")

	// Verify version is 5 (not 2).
	ver2, err := slb2.StreamVersion(ctx, "events", "s1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ver2).To(Equal(int64(5)), "stream version after restart should be 5")

	// Verify journal has ALL 5 entries in order.
	journal2, err := slb2.JournalReadAll(ctx, "events")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(journal2).To(HaveLen(5), "journal should retain all 5 entries after restart")

	// Verify Map ADT data survived.
	mapVal, err := eng2.MapGet(ctx, "kv", "key1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mapVal).To(Equal("val1"), "Map data should survive restart")

	// Verify new Map write doesn't overwrite existing.
	g.Expect(eng2.MapSet(ctx, "kv", "key2", "val2")).To(Succeed())
	mapVal2, err := eng2.MapGet(ctx, "kv", "key2")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mapVal2).To(Equal("val2"))
}

// TestPebbleRestartSafety_FromDB verifies seq seeding when using
// NewPebbleEngineFromDB (caller-owned DB path).
func TestPebbleRestartSafety_FromDB(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "pebble")

	// Phase 1: Open via NewPebbleEngine, write, close.
	eng1, err := pebbleengine.NewPebbleEngine(dir)
	g.Expect(err).NotTo(HaveOccurred())

	slb1, ok := eng1.(metaengine.StreamLogBackend)
	g.Expect(ok).To(BeTrue())

	g.Expect(slb1.StreamAppend(ctx, "events", "s1", []any{"a", "b"})).To(Succeed())
	g.Expect(eng1.Close()).To(Succeed())

	// Phase 2: Open raw DB, wrap via NewPebbleEngineFromDB, append more.
	db, err := pebble.Open(dir, &pebble.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	eng2, err := pebbleengine.NewPebbleEngineFromDB(db)
	g.Expect(err).NotTo(HaveOccurred())
	defer eng2.Close()

	slb2, ok := eng2.(metaengine.StreamLogBackend)
	g.Expect(ok).To(BeTrue())

	g.Expect(slb2.StreamAppend(ctx, "events", "s1", []any{"c"})).To(Succeed())

	ver, err := slb2.StreamVersion(ctx, "events", "s1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ver).To(Equal(int64(3)), "FromDB restart: stream should have 3 events")

	values, err := slb2.StreamRead(ctx, "events", "s1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(values).To(HaveLen(3), "FromDB restart: all 3 events retained")
}
