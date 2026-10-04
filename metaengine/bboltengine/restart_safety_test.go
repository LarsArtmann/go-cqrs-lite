package bboltengine_test

import (
	"context"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	bolt "go.etcd.io/bbolt"

	"github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestBboltRestartSafety_StreamAndJournal verifies that reopening a persistent
// bbolt DB does NOT reset seq counters to zero — which would cause silent key
// collisions and data loss (see enginetest.RunRestartSafetyTest).
func TestBboltRestartSafety_StreamAndJournal(t *testing.T) {
	t.Parallel()

	enginetest.RunRestartSafetyTest(t, func(path string) (metaengine.Engine, error) {
		return bboltengine.NewBboltEngine(path)
	})
}

// TestBboltRestartSafety_FromDB verifies seq seeding when using
// NewBboltEngineFromDB (caller-owned DB path).
func TestBboltRestartSafety_FromDB(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "bbolt.db")

	// Phase 1: Open via NewBboltEngine, write, close.
	eng1, err := bboltengine.NewBboltEngine(dir)
	g.Expect(err).NotTo(HaveOccurred())

	slb1, ok := eng1.(metaengine.StreamLogBackend)
	g.Expect(ok).To(BeTrue())

	g.Expect(slb1.StreamAppend(ctx, "events", "s1", []any{"a", "b"})).To(Succeed())
	g.Expect(eng1.Close()).To(Succeed())

	// Phase 2: Open raw DB, wrap via NewBboltEngineFromDB, append more.
	db, err := bolt.Open(dir, 0o600, bolt.DefaultOptions)
	g.Expect(err).NotTo(HaveOccurred())

	eng2, err := bboltengine.NewBboltEngineFromDB(db)
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
