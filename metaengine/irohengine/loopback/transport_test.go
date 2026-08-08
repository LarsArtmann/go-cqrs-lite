package loopback_test

import (
	"context"
	"strconv"
	"sync"
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

func setupTwoNodeLoopback(
	t *testing.T,
) (nodeA, nodeB metaengine.Engine, tA, tB *loopback.LoopbackTransport) {
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

func TestLoopbackConvergenceSuite(t *testing.T) {
	t.Parallel()
	irohengine.RunConvergenceSuite(t, func(t *testing.T) (metaengine.Engine, metaengine.Engine) {
		nodeA, nodeB, _, _ := setupTwoNodeLoopback(t)
		t.Cleanup(func() { _ = nodeA.Close() })
		t.Cleanup(func() { _ = nodeB.Close() })
		return nodeA, nodeB
	})
}

func TestLoopbackLWWConvergence(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()
	nodeA, nodeB, _, _ := setupTwoNodeLoopback(t)

	g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "lww", "key", "first")).To(gomega.Succeed())
	time.Sleep(50 * time.Millisecond)
	g.Expect(nodeB.(metaengine.MapBackend).MapSet(ctx, "lww", "key", "second")).To(gomega.Succeed())

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

	var mu sync.Mutex
	var received []irohengine.WriteOp
	g.Expect(tB.Subscribe(func(op irohengine.WriteOp) {
		mu.Lock()
		received = append(received, op)
		mu.Unlock()
	})).To(gomega.Succeed())

	g.Expect(tB.Connect(tA.Addr())).To(gomega.Succeed())
	waitForPeers(t, []*loopback.LoopbackTransport{tA, tB}, 1)

	ops := []irohengine.WriteOp{
		{ID: "op1", Collection: "c", Kind: irohengine.OpMapSet, Key: "k", Value: "v"},
		{
			ID:         "op2",
			Collection: "c",
			Kind:       irohengine.OpCounterInc,
			Delta:      metaengine.Delta{"alice": 5},
		},
		{ID: "op3", Collection: "c", Kind: irohengine.OpSetAdd, Key: "set1", Value: "tag1"},
	}

	for _, op := range ops {
		g.Expect(tA.Publish(context.Background(), op)).To(gomega.Succeed())
	}

	g.Eventually(func(g gomega.Gomega) {
		mu.Lock()
		defer mu.Unlock()
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
		key := "key-" + strconv.Itoa(i)
		val := "val-" + strconv.Itoa(i)
		g.Expect(mb.MapSet(ctx, "bulk", key, val)).To(gomega.Succeed())
	}

	g.Eventually(func(g gomega.Gomega) {
		mb2 := nodeB.(metaengine.MapBackend)
		for i := range opCount {
			val, ok, err := mb2.MapGet(ctx, "bulk", "key-"+strconv.Itoa(i))
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(ok).To(gomega.BeTrue())
			g.Expect(val).To(gomega.Equal("val-" + strconv.Itoa(i)))
		}
	}, 10*time.Second, 100*time.Millisecond).Should(gomega.Succeed())
}

func TestLoopbackLatencyMeasurement(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()
	nodeA, _, tA, _ := setupTwoNodeLoopback(t)

	mb := nodeA.(metaengine.MapBackend)
	g.Expect(mb.MapSet(ctx, "c", "k", "v")).To(gomega.Succeed())

	time.Sleep(500 * time.Millisecond)

	snap := tA.LatencySnapshot()
	_ = snap // just verify it doesn't panic
}


