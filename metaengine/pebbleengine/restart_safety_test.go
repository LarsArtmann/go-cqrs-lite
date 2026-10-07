package pebbleengine_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/cockroachdb/pebble"
	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
	. "github.com/onsi/gomega"
)

// TestPebbleRestartSafety_StreamAndJournal verifies that reopening a persistent
// Pebble DB does NOT reset seq counters to zero — which would cause silent key
// collisions and data loss (see enginetest.RunRestartSafetyTest).
func TestPebbleRestartSafety_StreamAndJournal(t *testing.T) {
	t.Parallel()

	enginetest.RunRestartSafetyTest(t, func(path string) (metaengine.Engine, error) {
		return pebbleengine.NewPebbleEngine(path)
	})
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
