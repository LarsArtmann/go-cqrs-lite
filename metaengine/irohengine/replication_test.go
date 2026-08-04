package irohengine_test

import (
	"context"
	"testing"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestMapUpdateDoesNotReplicate proves that MapUpdate (atomic read-modify-write)
// executes locally but does NOT cross node boundaries. This is the CALM theorem
// constraint: non-monotonic operations cannot converge via CRDT.
func TestMapUpdateDoesNotReplicate(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	net := irohengine.NewNetwork()
	nodeA := irohengine.Replicated(metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-a"),
		irohengine.WithTransport(net.Join("a")),
	)
	nodeB := irohengine.Replicated(metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-b"),
		irohengine.WithTransport(net.Join("b")),
	)
	defer nodeA.Close()
	defer nodeB.Close()

	g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "counters", "c1", 0)).To(gomega.Succeed())

	g.Expect(nodeA.(metaengine.MapUpdater).MapUpdate(ctx, "counters", "c1", func(prev any) any {
		if n, ok := prev.(int); ok {
			return n + 1
		}
		return 1
	})).To(gomega.Succeed())

	valA, _, err := nodeA.(metaengine.MapBackend).MapGet(ctx, "counters", "c1")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(valA).To(gomega.Equal(1), "local MapUpdate applied")

	valB, okB, err := nodeB.(metaengine.MapBackend).MapGet(ctx, "counters", "c1")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(okB).To(gomega.BeTrue(), "MapSet replicated")
	g.Expect(valB).To(gomega.Equal(0), "MapUpdate must NOT replicate (non-CRDT)")
}

func TestNilTransport(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	eng := irohengine.Replicated(metaengine.NewMemoryEngine())
	defer eng.Close()

	g.Expect(eng.(metaengine.MapBackend).MapSet(ctx, "x", "k", "v")).To(gomega.Succeed())
	val, ok, err := eng.(metaengine.MapBackend).MapGet(ctx, "x", "k")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(val).To(gomega.Equal("v"))
}
