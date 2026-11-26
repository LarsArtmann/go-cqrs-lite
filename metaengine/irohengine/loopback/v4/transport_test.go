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
) (metaengine.Engine, metaengine.Engine, *loopback.LoopbackTransport, *loopback.LoopbackTransport) {
	t.Helper()
	g := gomega.NewWithT(t)

	transportA, err := loopback.New()
	g.Expect(err).NotTo(gomega.HaveOccurred())

	transportB, err := loopback.New()
	g.Expect(err).NotTo(gomega.HaveOccurred())

	engineA := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-a"),
		irohengine.WithTransport(transportA),
	)
	engineB := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-b"),
		irohengine.WithTransport(transportB),
	)

	g.Expect(transportB.Connect(transportA.Addr())).To(gomega.Succeed())
	waitForPeers(t, []*loopback.LoopbackTransport{transportA, transportB}, 1)

	t.Cleanup(func() {
		_ = transportA.Close()
		_ = transportB.Close()
	})

	return engineA, engineB, transportA, transportB
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

	transportA, err := loopback.New()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	t.Cleanup(func() { _ = transportA.Close() })

	transportB, err := loopback.New()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	t.Cleanup(func() { _ = transportB.Close() })

	var mu sync.Mutex
	var received []irohengine.WriteOp
	g.Expect(transportB.Subscribe(func(op irohengine.WriteOp) {
		mu.Lock()
		received = append(received, op)
		mu.Unlock()
	})).To(gomega.Succeed())

	g.Expect(transportB.Connect(transportA.Addr())).To(gomega.Succeed())
	waitForPeers(t, []*loopback.LoopbackTransport{transportA, transportB}, 1)

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
		g.Expect(transportA.Publish(context.Background(), op)).To(gomega.Succeed())
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

// Concurrent publishers share one TCP connection; without a per-connection
// write lock their frames interleave and the peer's reader loses sync.
func TestLoopbackConcurrentPublishKeepsFramesWhole(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	transportA, err := loopback.New()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	t.Cleanup(func() { _ = transportA.Close() })

	transportB, err := loopback.New()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	t.Cleanup(func() { _ = transportB.Close() })

	const publishers, opsEach = 8, 200

	var (
		mu       sync.Mutex
		received int
	)
	g.Expect(transportB.Subscribe(func(irohengine.WriteOp) {
		mu.Lock()
		received++
		mu.Unlock()
	})).To(gomega.Succeed())

	g.Expect(transportB.Connect(transportA.Addr())).To(gomega.Succeed())
	waitForPeers(t, []*loopback.LoopbackTransport{transportA}, 1)

	var wg sync.WaitGroup
	for p := range publishers {
		wg.Add(1)

		go func(p int) {
			defer wg.Done()

			for i := range opsEach {
				op := irohengine.WriteOp{
					ID:         "conc-" + strconv.Itoa(p) + "-" + strconv.Itoa(i),
					Collection: "conc",
					Kind:       irohengine.OpMapSet,
					Key:        "k",
					Value:      "payload-that-makes-frames-span-multiple-writes-" + strconv.Itoa(i),
				}
				_ = transportA.Publish(context.Background(), op)
			}
		}(p)
	}
	wg.Wait()

	g.Eventually(func(g gomega.Gomega) {
		mu.Lock()
		defer mu.Unlock()

		g.Expect(received).To(gomega.Equal(publishers * opsEach))
	}, 10*time.Second, 50*time.Millisecond).Should(gomega.Succeed())
}
