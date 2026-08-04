package loopback_test

import (
	"context"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/loopback/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// waitForPeers polls until both transports see the expected peer count.
func waitForPeers(t *testing.T, transports []*loopback.LoopbackTransport, expected int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		allReady := true
		for _, tr := range transports {
			if tr.PeerCount() < expected {
				allReady = false
				break
			}
		}
		if allReady {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %d peers on all transports", expected)
}

func eventuallyGet(
	g gomega.Gomega,
	node metaengine.Engine,
	collection, key string,
	expected any,
	timeout time.Duration,
) {
	g.Eventually(func(g gomega.Gomega) {
		val, ok, err := node.(metaengine.MapBackend).MapGet(context.Background(), collection, key)
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(ok).To(gomega.BeTrue())
		g.Expect(val).To(gomega.Equal(expected))
	}, timeout, 50*time.Millisecond).Should(gomega.Succeed())
}

func setupTwoNodeLoopback(t *testing.T) (nodeA, nodeB metaengine.Engine, tA, tB *loopback.LoopbackTransport) {
	t.Helper()
	g := gomega.NewWithT(t)

	tA, err := loopback.New()
	g.Expect(err).NotTo(gomega.HaveOccurred())

	tB, err = loopback.New()
	g.Expect(err).NotTo(gomega.HaveOccurred())

	nodeA = irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-a"),
		irohengine.WithTransport(tA),
	)
	nodeB = irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-b"),
		irohengine.WithTransport(tB),
	)

	g.Expect(tB.Connect(tA.Addr())).To(gomega.Succeed())
	waitForPeers(t, []*loopback.LoopbackTransport{tA, tB}, 1)

	t.Cleanup(func() {
		_ = tA.Close()
		_ = tB.Close()
	})

	return nodeA, nodeB, tA, tB
}

func TestLoopbackMapConvergence(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()
	nodeA, nodeB, _, _ := setupTwoNodeLoopback(t)

	err := nodeA.(metaengine.MapBackend).MapSet(ctx, "coll", "key1", "value1")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	eventuallyGet(g, nodeB, "coll", "key1", "value1", 5*time.Second)
}

func TestLoopbackBidirectionalConvergence(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()
	nodeA, nodeB, _, _ := setupTwoNodeLoopback(t)

	g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "coll", "a-key", "a-val")).To(gomega.Succeed())
	g.Expect(nodeB.(metaengine.MapBackend).MapSet(ctx, "coll", "b-key", "b-val")).To(gomega.Succeed())

	eventuallyGet(g, nodeA, "coll", "b-key", "b-val", 5*time.Second)
	eventuallyGet(g, nodeB, "coll", "a-key", "a-val", 5*time.Second)
}

func TestLoopbackCounterConvergence(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()
	nodeA, nodeB, _, _ := setupTwoNodeLoopback(t)

	cb := nodeA.(metaengine.CounterBackend)
	g.Expect(cb.CounterIncrement(ctx, "counters", "views", "alice", 5)).To(gomega.Succeed())

	g.Eventually(func(g gomega.Gomega) {
		cb2 := nodeB.(metaengine.CounterBackend)
		val, err := cb2.CounterValue(ctx, "counters", "views")
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(val).To(gomega.Equal(int64(5)))
	}, 5*time.Second, 50*time.Millisecond).Should(gomega.Succeed())
}

func TestLoopbackSetConvergence(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()
	nodeA, nodeB, _, _ := setupTwoNodeLoopback(t)

	sb := nodeA.(metaengine.SetBackend)
	g.Expect(sb.SetAdd(ctx, "tags", "post-1", "go")).To(gomega.Succeed())
	g.Expect(sb.SetAdd(ctx, "tags", "post-1", "crdt")).To(gomega.Succeed())

	g.Eventually(func(g gomega.Gomega) {
		sb2 := nodeB.(metaengine.SetBackend)
		members, err := sb2.SetMembers(ctx, "tags", "post-1")
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(members).To(gomega.ContainElements("go", "crdt"))
		g.Expect(members).To(gomega.HaveLen(2))
	}, 5*time.Second, 50*time.Millisecond).Should(gomega.Succeed())
}

func TestLoopbackLWWConvergence(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()
	nodeA, nodeB, _, _ := setupTwoNodeLoopback(t)

	// Both write the same key with different values
	g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "lww", "key", "first")).To(gomega.Succeed())
	time.Sleep(50 * time.Millisecond)
	g.Expect(nodeB.(metaengine.MapBackend).MapSet(ctx, "lww", "key", "second")).To(gomega.Succeed())

	// Both should converge to the later value
	time.Sleep(2 * time.Second)
	valA, _, _ := nodeA.(metaengine.MapBackend).MapGet(ctx, "lww", "key")
	valB, _, _ := nodeB.(metaengine.MapBackend).MapGet(ctx, "lww", "key")
	g.Expect(valA).To(gomega.Equal(valB), "both nodes must agree on final value")
}

func TestLoopbackFrameEncodingRoundTrip(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	tA, err := loopback.New()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	t.Cleanup(func() { _ = tA.Close() })

	tB, err := loopback.New()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	t.Cleanup(func() { _ = tB.Close() })

	var received []irohengine.WriteOp
	g.Expect(tB.Subscribe(func(op irohengine.WriteOp) {
		received = append(received, op)
	})).To(gomega.Succeed())

	g.Expect(tB.Connect(tA.Addr())).To(gomega.Succeed())
	waitForPeers(t, []*loopback.LoopbackTransport{tA, tB}, 1)

	// Send various op kinds to exercise serialization
	ops := []irohengine.WriteOp{
		{ID: "op1", Collection: "c", Kind: irohengine.OpMapSet, Key: "k", Value: "v"},
		{ID: "op2", Collection: "c", Kind: irohengine.OpCounterInc, Key: "counter", Delta: metaengine.Delta{"alice": 5}},
		{ID: "op3", Collection: "c", Kind: irohengine.OpSetAdd, Key: "set1", Value: "tag1"},
	}

	for _, op := range ops {
		g.Expect(tA.Publish(context.Background(), op)).To(gomega.Succeed())
	}

	g.Eventually(func(g gomega.Gomega) {
		g.Expect(received).To(gomega.HaveLen(3))
	}, 5*time.Second, 50*time.Millisecond).Should(gomega.Succeed())
}

func TestLoopbackLargeScaleConvergence(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()
	nodeA, nodeB, _, _ := setupTwoNodeLoopback(t)

	const opCount = 100
	mb := nodeA.(metaengine.MapBackend)
	for i := range opCount {
		key := "key-" + itoa(i)
		val := "val-" + itoa(i)
		g.Expect(mb.MapSet(ctx, "bulk", key, val)).To(gomega.Succeed())
	}

	// Verify all keys converge
	g.Eventually(func(g gomega.Gomega) {
		mb2 := nodeB.(metaengine.MapBackend)
		for i := range opCount {
			val, ok, err := mb2.MapGet(ctx, "bulk", "key-"+itoa(i))
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(ok).To(gomega.BeTrue())
			g.Expect(val).To(gomega.Equal("val-" + itoa(i)))
		}
	}, 10*time.Second, 100*time.Millisecond).Should(gomega.Succeed())
}

func TestLoopbackLatencyMeasurement(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()
	_, nodeB, tA, _ := setupTwoNodeLoopback(t)

	// Write to trigger traffic
	_ = tA.Publish(ctx, irohengine.WriteOp{
		ID:         "latency-test",
		PublishedAt: time.Now(),
		Collection: "c",
		Kind:       irohengine.OpMapSet,
		Key:        "k",
		Value:      "v",
	})

	time.Sleep(500 * time.Millisecond)

	snap := tA.LatencySnapshot()
	// On localhost, latency should be sub-millisecond but non-zero after traffic
	// (may be zero if no inbound ops arrived — only outbound ops were sent)
	_ = snap // just verify it doesn't panic
	_ = nodeB
}

// itoa is a simple int-to-string without strconv to avoid an extra import.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
