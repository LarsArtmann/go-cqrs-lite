package pebbleengine_test

import (
	"context"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// disk_backed_test.go verifies the on-disk mode of NewPebbleEngine(dir): data
// must survive Close→reopen when a real directory is used (previously the dir
// argument was ignored and pebble.Open received "" — a silent bug this test
// pins). In-memory mode (dir=="") is intentionally non-persistent.

func TestPebbleDiskBacked_PersistsAcrossReopen(t *testing.T) {
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "pebble")
	ctx := context.Background()

	// First open: write, then close.
	eng1, err := pebbleengine.NewPebbleEngine(dbPath)
	g.Expect(err).NotTo(HaveOccurred())

	mb1 := eng1.(metaengine.MapBackend)
	g.Expect(mb1.MapSet(ctx, "users", "u1", map[string]any{"name": "Alice"})).To(Succeed())
	g.Expect(eng1.Close()).To(Succeed())

	// Second open of the SAME directory: the row must survive.
	eng2, err := pebbleengine.NewPebbleEngine(dbPath)
	g.Expect(err).NotTo(HaveOccurred())

	defer eng2.Close()

	mb2 := eng2.(metaengine.MapBackend)
	val, found, err := mb2.MapGet(ctx, "users", "u1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())
	g.Expect(val).To(Equal(map[string]any{"name": "Alice"}))
}

func TestPebbleInMemory_DoesNotPersist(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx := context.Background()

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())

	mb := eng.(metaengine.MapBackend)
	g.Expect(mb.MapSet(ctx, "users", "u1", "Alice")).To(Succeed())
	g.Expect(eng.Close()).To(Succeed())

	// A fresh in-memory engine is always empty.
	eng2, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())

	defer eng2.Close()

	mb2 := eng2.(metaengine.MapBackend)
	_, found, err := mb2.MapGet(ctx, "users", "u1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeFalse())
}
