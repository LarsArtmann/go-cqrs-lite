package pebbleengine_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// pebbleengine_test.go validates the Pebble engine's basic operations and
// cross-engine behavioral parity with the memory and SQLite engines.

func TestPebbleMapSetGet(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)

	g.Expect(mb.MapSet(ctx, "users", "u1", map[string]any{"name": "Alice"})).To(Succeed())

	val, found, err := mb.MapGet(ctx, "users", "u1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())
	g.Expect(val).To(Equal(map[string]any{"name": "Alice"}))
}

func TestPebbleMapDelete(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)

	g.Expect(mb.MapSet(ctx, "users", "u1", "Alice")).To(Succeed())
	g.Expect(mb.MapDelete(ctx, "users", "u1")).To(Succeed())

	_, found, err := mb.MapGet(ctx, "users", "u1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeFalse())
}

func TestPebbleSetAddContains(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	ctx := context.Background()
	sb := eng.(metaengine.SetBackend)

	g.Expect(sb.SetAdd(ctx, "tags", "go")).To(Succeed())
	g.Expect(sb.SetAdd(ctx, "tags", "rust")).To(Succeed())

	contains, err := sb.SetContains(ctx, "tags", "go")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(contains).To(BeTrue())

	contains, err = sb.SetContains(ctx, "tags", "python")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(contains).To(BeFalse())
}

func TestPebbleCounter(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	ctx := context.Background()
	cb := eng.(metaengine.CounterBackend)

	g.Expect(cb.CounterIncrement(ctx, "counts", metaengine.Delta{"open": +3})).To(Succeed())
	g.Expect(cb.CounterIncrement(ctx, "counts", metaengine.Delta{"open": +2, "done": +1})).To(Succeed())

	counts, err := cb.CounterGet(ctx, "counts")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(counts["open"]).To(Equal(int64(5)))
	g.Expect(counts["done"]).To(Equal(int64(1)))
}

func TestPebbleMultimap(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	ctx := context.Background()
	mb := eng.(metaengine.MultimapBackend)

	g.Expect(mb.MultiAdd(ctx, "tasks_by_user", "alice", "task1")).To(Succeed())
	g.Expect(mb.MultiAdd(ctx, "tasks_by_user", "alice", "task2")).To(Succeed())
	g.Expect(mb.MultiAdd(ctx, "tasks_by_user", "bob", "task3")).To(Succeed())

	aliceTasks, err := mb.MultiGet(ctx, "tasks_by_user", "alice")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(aliceTasks).To(HaveLen(2))
	g.Expect(aliceTasks[0]).To(Equal("task1"))
	g.Expect(aliceTasks[1]).To(Equal("task2"))
}

func TestPebbleLog(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	ctx := context.Background()
	lb := eng.(metaengine.LogBackend)

	g.Expect(lb.LogAppend(ctx, "audit", "event1")).To(Succeed())
	g.Expect(lb.LogAppend(ctx, "audit", "event2")).To(Succeed())
	g.Expect(lb.LogAppend(ctx, "audit", "event3")).To(Succeed())

	tail, err := lb.LogTail(ctx, "audit", 2)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(tail).To(HaveLen(2))
	g.Expect(tail[0]).To(Equal("event2"))
	g.Expect(tail[1]).To(Equal("event3"))

	all, err := lb.LogTail(ctx, "audit", 0)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(all).To(HaveLen(3))
}

func TestPebbleGraphNeighbors(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	ctx := context.Background()
	gb := eng.(metaengine.GraphBackend)

	g.Expect(gb.GraphAddEdge(ctx, "social", metaengine.Edge{From: "alice", To: "bob"})).To(Succeed())
	g.Expect(gb.GraphAddEdge(ctx, "social", metaengine.Edge{From: "bob", To: "carol"})).To(Succeed())

	neighbors, err := gb.GraphNeighbors(ctx, "social", "alice", 1)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(neighbors).To(ContainElement("bob"))

	neighbors2, err := gb.GraphNeighbors(ctx, "social", "alice", 2)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(neighbors2).To(ContainElement("bob"))
	g.Expect(neighbors2).To(ContainElement("carol"))
}

func TestPebbleMapUpdate(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	ctx := context.Background()
	mu := eng.(metaengine.MapUpdater)
	mb := eng.(metaengine.MapBackend)

	g.Expect(mb.MapSet(ctx, "counters", "c1", float64(5))).To(Succeed())

	g.Expect(mu.MapUpdate(ctx, "counters", "c1", func(prev any) any {
		if p, ok := prev.(float64); ok {
			return p + 1
		}

		return float64(1)
	})).To(Succeed())

	val, found, err := mb.MapGet(ctx, "counters", "c1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())
	g.Expect(val).To(Equal(float64(6)))
}

func TestPebbleMapScan(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)
	sb := eng.(metaengine.ScanBackend)

	for i := 0; i < 10; i++ {
		g.Expect(mb.MapSet(ctx, "items", i, map[string]any{
			"ID":     i,
			"Status": "open",
		})).To(Succeed())
	}

	// No filter — all 10 items.
	results, err := sb.MapScan(ctx, "items", nil, nil, nil, 0)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(results).To(HaveLen(10))

	// With filter — only matching items.
	filterFn := func(item any) bool {
		m, ok := item.(map[string]any)
		if !ok {
			return false
		}

		id, ok := m["ID"].(float64)
		return ok && id < 5
	}

	results, err = sb.MapScan(ctx, "items", filterFn, nil, nil, 0)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(results).To(HaveLen(5))
}

func TestPebbleProfile(t *testing.T) {
	g := NewGomegaWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(HaveOccurred())
	defer eng.Close()

	p := eng.Profile()
	g.Expect(p.Name).To(Equal("pebble"))
	g.Expect(p.Supports).To(HaveKey(metaengine.ADTMap))
	g.Expect(p.Supports[metaengine.ADTMap]).To(Equal(metaengine.ComplexityO1))
}
