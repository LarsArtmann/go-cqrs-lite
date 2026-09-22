package bboltengine_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	. "github.com/onsi/gomega"
)

// disk_backed_test.go verifies the on-disk mode of NewBboltEngine(path): data
// must survive Close→reopen when a real path is used. In-memory mode
// (path=="") uses a temp file that is deleted on Close, so it is intentionally
// non-persistent across engine instances.

func TestBboltDiskBacked_PersistsAcrossReopen(t *testing.T) {
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "bbolt.db")
	ctx := context.Background()

	// First open: write, then close.
	eng1, err := bboltengine.NewBboltEngine(dbPath)
	g.Expect(err).NotTo(HaveOccurred())

	mb1 := eng1.(metaengine.MapBackend)
	g.Expect(mb1.MapSet(ctx, "users", "u1", map[string]any{"name": "Alice"})).To(Succeed())
	g.Expect(eng1.Close()).To(Succeed())

	// Second open of the SAME path: the row must survive.
	eng2, err := bboltengine.NewBboltEngine(dbPath)
	g.Expect(err).NotTo(HaveOccurred())

	defer eng2.Close()

	mb2 := eng2.(metaengine.MapBackend)
	val, found, err := mb2.MapGet(ctx, "users", "u1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())
	g.Expect(val).To(Equal(map[string]any{"name": "Alice"}))
}

func TestBboltInMemory_DoesNotPersist(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx := context.Background()

	eng, err := bboltengine.NewBboltEngine("")
	g.Expect(err).NotTo(HaveOccurred())

	mb := eng.(metaengine.MapBackend)
	g.Expect(mb.MapSet(ctx, "users", "u1", "Alice")).To(Succeed())
	g.Expect(eng.Close()).To(Succeed())

	// A fresh in-memory engine is always empty (new temp file).
	eng2, err := bboltengine.NewBboltEngine("")
	g.Expect(err).NotTo(HaveOccurred())

	defer eng2.Close()

	mb2 := eng2.(metaengine.MapBackend)
	_, found, err := mb2.MapGet(ctx, "users", "u1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeFalse())
}
