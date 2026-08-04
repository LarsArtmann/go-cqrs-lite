package irohengine_test

import (
	"context"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestLatencyMeasuredFromRealTraffic(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	net := irohengine.NewNetwork(
		irohengine.WithNetworkDelay(15 * time.Millisecond),
	)
	nodeA := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("a"),
		irohengine.WithTransport(net.Join("a")),
	)
	nodeB := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("b"),
		irohengine.WithTransport(net.Join("b")),
	)
	defer nodeA.Close()
	defer nodeB.Close()

	// Profile BEFORE traffic: zero, not hardcoded
	p0 := nodeA.Profile()
	g.Expect(p0.ReplicationLag).To(gomega.BeZero(), "lag must be zero before any traffic")
	g.Expect(p0.NetworkRTT).To(gomega.BeZero(), "rtt must be zero before any traffic")

	// Generate traffic
	for i := 0; i < 30; i++ {
		g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "k", i, i)).To(gomega.Succeed())
	}

	// Profile AFTER traffic: measured
	p1 := nodeA.Profile()
	g.Expect(p1.ReplicationLag).
		To(gomega.BeNumerically(">", 0), "lag must be measured after traffic")
	g.Expect(p1.NetworkRTT).To(gomega.BeNumerically(">", 0), "rtt must be measured after traffic")

	// With 15ms max delay, P99 should be within a reasonable bound
	g.Expect(p1.ReplicationLag).To(gomega.BeNumerically("<=", 50*time.Millisecond),
		"lag should be bounded by network delay")

	// Delivery stats should have real samples
	c := net.Collector()
	dStats := c.DeliveryStats()
	g.Expect(dStats.Samples).
		To(gomega.Equal(30), "one delivery sample per op (2 peers, but only 1 other node)")

	convStats := c.ConvergenceStats()
	g.Expect(convStats.Samples).To(gomega.Equal(30), "one convergence sample per op")
	g.Expect(convStats.P99).To(gomega.BeNumerically(">=", 1*time.Millisecond),
		"convergence P99 should reflect real delay")
}

func TestLatencyScalesWithDelay(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		delay    time.Duration
		maxBound time.Duration
	}{
		{"fast", 1 * time.Millisecond, 20 * time.Millisecond},
		{"medium", 20 * time.Millisecond, 100 * time.Millisecond},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			ctx := context.Background()

			net := irohengine.NewNetwork(
				irohengine.WithNetworkDelay(tc.delay),
			)
			nodeA := irohengine.Replicated(
				metaengine.NewMemoryEngine(),
				irohengine.WithAuthor("a"),
				irohengine.WithTransport(net.Join("a")),
			)
			nodeB := irohengine.Replicated(
				metaengine.NewMemoryEngine(),
				irohengine.WithAuthor("b"),
				irohengine.WithTransport(net.Join("b")),
			)
			defer nodeA.Close()
			defer nodeB.Close()

			for i := 0; i < 20; i++ {
				_ = nodeA.(metaengine.MapBackend).MapSet(ctx, "x", i, i)
			}

			conv := net.Collector().ConvergenceStats()
			g.Expect(conv.Mean).To(gomega.BeNumerically(">", 0),
				"convergence mean must be positive")
			g.Expect(conv.Mean).To(gomega.BeNumerically("<", tc.maxBound),
				"convergence mean %s should be bounded by %s for %s delay", conv.Mean, tc.maxBound, tc.delay)
		})
	}
}

func TestProfileReflectsRealRTT(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	// With 10ms delay, RTT (2× P50 delivery) should be in a reasonable range
	net := irohengine.NewNetwork(
		irohengine.WithNetworkDelay(10 * time.Millisecond),
	)
	nodeA := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("a"),
		irohengine.WithTransport(net.Join("a")),
	)
	nodeB := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("b"),
		irohengine.WithTransport(net.Join("b")),
	)
	defer nodeA.Close()
	defer nodeB.Close()

	for i := 0; i < 50; i++ {
		_ = nodeA.(metaengine.MapBackend).MapSet(ctx, "rtt-test", i, i)
	}

	p := nodeA.Profile()
	// RTT = 2 × P50 delivery. With 0-10ms delay, P50 ~5ms, RTT ~10ms
	g.Expect(p.NetworkRTT).To(gomega.BeNumerically(">", 2*time.Millisecond),
		"RTT must reflect measured delay")
	g.Expect(p.NetworkRTT).To(gomega.BeNumerically("<", 30*time.Millisecond),
		"RTT must be bounded")
}

func TestConcurrentWritesAllConverge(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	net := irohengine.NewNetwork(
		irohengine.WithNetworkDelay(5 * time.Millisecond),
	)

	nodes := make([]metaengine.Engine, 3)
	for i, id := range []string{"a", "b", "c"} {
		nodes[i] = irohengine.Replicated(
			metaengine.NewMemoryEngine(),
			irohengine.WithAuthor(id),
			irohengine.WithTransport(net.Join(id)),
		)
		defer nodes[i].Close()
	}

	// Each node writes 30 keys concurrently (keys 0-29, 30-59, 60-89)
	type result struct {
		node int
		err  error
	}
	done := make(chan result, 90)
	for ni, n := range nodes {
		go func(ni int, n metaengine.Engine) {
			for i := 0; i < 30; i++ {
				err := n.(metaengine.MapBackend).MapSet(ctx, "storm", ni*30+i, i)
				done <- result{ni, err}
			}
		}(ni, n)
	}

	for i := 0; i < 90; i++ {
		r := <-done
		g.Expect(r.err).NotTo(gomega.HaveOccurred())
	}

	// Verify all 90 keys present on all nodes
	for ni, n := range nodes {
		for k := 0; k < 90; k++ {
			_, ok, err := n.(metaengine.MapBackend).MapGet(ctx, "storm", k)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(ok).To(gomega.BeTrue(), "node %d missing key %d", ni, k)
		}
	}
}
