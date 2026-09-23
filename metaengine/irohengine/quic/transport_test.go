//go:build cgo

package quic_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/quic/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/onsi/gomega"
)

// quicManualClock is a deterministic Clock for QUIC tests. It starts at a
// fixed epoch and only advances when Advance is called — eliminating all
// timing assumptions in LWW convergence tests. Mirrors the in-process
// manualClock in irohengine/helpers_test.go.
type quicManualClock struct {
	now atomic.Int64 // unix-nanos
}

func newQuicManualClock(start time.Time) *quicManualClock {
	c := &quicManualClock{}
	c.now.Store(start.UnixNano())
	return c
}

func (c *quicManualClock) Now() time.Time {
	return time.Unix(0, c.now.Load())
}

// Advance moves the clock forward by d and returns the new time.
func (c *quicManualClock) Advance(d time.Duration) time.Time {
	return time.Unix(0, c.now.Add(int64(d)))
}

// waitForPeers polls until both transports see the expected peer count, or times out.
func waitForPeers(t *testing.T, transports []*quic.QuicTransport, expected int) {
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

// eventuallyGet retries a MapGet until it matches or times out.
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

// quicCluster bundles the two engines + transports + gomega + context for a
// single convergence test. Setup cleanup is wired via t.Cleanup.
type quicCluster struct {
	G     gomega.Gomega
	Ctx   context.Context //nolint:containedctx // test helper
	NodeA metaengine.Engine
	NodeB metaengine.Engine
	TA    *quic.QuicTransport
	TB    *quic.QuicTransport
}

// setupTwoNodeQuic creates two connected QuicTransport nodes with engines.
func setupTwoNodeQuic(t *testing.T) (
	nodeA, nodeB metaengine.Engine,
	tA, tB *quic.QuicTransport,
) {
	t.Helper()
	g := gomega.NewWithT(t)

	tA, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())

	tB, err = quic.New(quic.WithLocalOnly())
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

	ticketA, err := tA.Ticket()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(tB.Connect(ticketA)).To(gomega.Succeed())

	waitForPeers(t, []*quic.QuicTransport{tA, tB}, 1)
	return nodeA, nodeB, tA, tB
}

// newQuicCluster is the per-test preamble: gomega + context + 2-node setup
// with cleanup deferred via t.Cleanup. Every test in this file uses the same
// preamble; the helper keeps each test focused on its ADT-specific assertions.
func newQuicCluster(t *testing.T) *quicCluster {
	t.Helper()

	nodeA, nodeB, tA, tB := setupTwoNodeQuic(t)
	t.Cleanup(func() { _ = nodeA.Close() })
	t.Cleanup(func() { _ = nodeB.Close() })
	t.Cleanup(func() { _ = tA.Close() })
	t.Cleanup(func() { _ = tB.Close() })

	return &quicCluster{
		G:     gomega.NewWithT(t),
		Ctx:   context.Background(),
		NodeA: nodeA,
		NodeB: nodeB,
		TA:    tA,
		TB:    tB,
	}
}

func TestQuicConvergenceSuite(t *testing.T) {
	t.Parallel()
	irohengine.RunConvergenceSuite(t, func(t *testing.T) (metaengine.Engine, metaengine.Engine) {
		nodeA, nodeB, tA, tB := setupTwoNodeQuic(t)
		t.Cleanup(func() { _ = nodeA.Close() })
		t.Cleanup(func() { _ = nodeB.Close() })
		t.Cleanup(func() { _ = tA.Close() })
		t.Cleanup(func() { _ = tB.Close() })
		return nodeA, nodeB
	})
}

func TestQuicLWWResolution(t *testing.T) {
	// Deterministic LWW test using injectable clock — same pattern as the
	// in-process TestLWWResolution. Both nodes share a quicManualClock, so
	// timestamp ordering is controlled by explicit Advance() calls instead
	// of relying on wall-clock time gaps.
	clock := newQuicManualClock(time.Unix(1_000_000, 0))

	t.Helper()
	g := gomega.NewWithT(t)

	tA, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	tB, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	t.Cleanup(func() { _ = tA.Close() })
	t.Cleanup(func() { _ = tB.Close() })

	nodeA := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-a"),
		irohengine.WithTransport(tA),
		irohengine.WithClock(clock),
	)
	nodeB := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-b"),
		irohengine.WithTransport(tB),
		irohengine.WithClock(clock),
	)
	t.Cleanup(func() { _ = nodeA.Close() })
	t.Cleanup(func() { _ = nodeB.Close() })

	ticketA, err := tA.Ticket()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(tB.Connect(ticketA)).To(gomega.Succeed())
	waitForPeers(t, []*quic.QuicTransport{tA, tB}, 1)

	ctx := context.Background()

	// NodeA writes first (timestamp T0 from the shared clock).
	g.Expect(nodeA.(metaengine.MapBackend).MapSet(ctx, "users", "u1", "Alice-old")).
		To(gomega.Succeed())
	eventuallyGet(g, nodeB, "users", "u1", "Alice-old", 5*time.Second)

	// Deterministic timestamp advance — no time.Sleep needed.
	// Node B's write gets a strictly later timestamp, guaranteeing LWW resolution.
	clock.Advance(time.Second)

	g.Expect(nodeB.(metaengine.MapBackend).MapSet(ctx, "users", "u1", "Bob-new")).
		To(gomega.Succeed())

	eventuallyGet(g, nodeA, "users", "u1", "Bob-new", 5*time.Second)
	eventuallyGet(g, nodeB, "users", "u1", "Bob-new", 5*time.Second)
}

func TestQuicRTTMeasurement(t *testing.T) {
	c := newQuicCluster(t)

	for range 10 {
		c.G.Expect(c.NodeA.(metaengine.MapBackend).MapSet(c.Ctx, "kvs",
			"key", "value")).To(gomega.Succeed())
	}

	profile := c.NodeA.Profile()
	t.Logf("Profile: ReplicationLag=%s NetworkRTT=%s", profile.ReplicationLag, profile.NetworkRTT)
	c.G.Expect(profile.Replication).To(gomega.Equal(metaengine.ReplicationLeaderless))
}

// TestQuicMapUpdateDoesNotReplicate proves that MapUpdate (atomic
// read-modify-write) executes locally but does NOT cross node boundaries over
// QUIC. This is the CALM theorem constraint: non-monotonic operations cannot
// converge via CRDT. Mirrors the in-process TestMapUpdateDoesNotReplicate but
// over the real QUIC transport.
func TestQuicMapUpdateDoesNotReplicate(t *testing.T) {
	c := newQuicCluster(t)

	c.G.Expect(c.NodeA.(metaengine.MapBackend).MapSet(c.Ctx, "counters", "c1", 0)).
		To(gomega.Succeed())

	// Wait for MapSet to replicate
	c.G.Eventually(func(g gomega.Gomega) {
		_, ok, err := c.NodeB.(metaengine.MapBackend).MapGet(c.Ctx, "counters", "c1")
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(ok).To(gomega.BeTrue())
	}, 5*time.Second, 50*time.Millisecond).Should(gomega.Succeed())

	c.G.Expect(c.NodeA.(metaengine.MapUpdater).MapUpdate(c.Ctx, "counters", "c1",
		func(prev any) any {
			n, _ := prev.(int)
			return n + 1
		})).To(gomega.Succeed())

	// Local MapUpdate applied
	c.G.Eventually(func(g gomega.Gomega) {
		valA, _, err := c.NodeA.(metaengine.MapBackend).MapGet(c.Ctx, "counters", "c1")
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(valA).To(gomega.Equal(1))
	}, 5*time.Second, 50*time.Millisecond).Should(gomega.Succeed())

	time.Sleep(500 * time.Millisecond) // give replication time if it (incorrectly) happened

	valB, okB, err := c.NodeB.(metaengine.MapBackend).MapGet(c.Ctx, "counters", "c1")
	c.G.Expect(err).NotTo(gomega.HaveOccurred())
	c.G.Expect(okB).To(gomega.BeTrue(), "MapSet replicated")
	c.G.Expect(valB).To(gomega.Equal(0),
		"MapUpdate must NOT replicate over QUIC (non-CRDT)")
}

// --- Stream Pooling Tests ---

// setupTwoNodeQuicPooled is like setupTwoNodeQuic but enables persistent
// BiStream pooling (WithStreamPooling) on both transports. This exercises the
// length-prefix framing protocol instead of the default one-stream-per-op mode.
func setupTwoNodeQuicPooled(t *testing.T) (
	nodeA, nodeB metaengine.Engine,
	tA, tB *quic.QuicTransport,
) {
	t.Helper()
	g := gomega.NewWithT(t)

	tA, err := quic.New(quic.WithLocalOnly(), quic.WithStreamPooling())
	g.Expect(err).NotTo(gomega.HaveOccurred())

	tB, err = quic.New(quic.WithLocalOnly(), quic.WithStreamPooling())
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

	ticketA, err := tA.Ticket()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(tB.Connect(ticketA)).To(gomega.Succeed())

	waitForPeers(t, []*quic.QuicTransport{tA, tB}, 1)
	return nodeA, nodeB, tA, tB
}

func newQuicClusterPooled(t *testing.T) *quicCluster {
	t.Helper()

	nodeA, nodeB, tA, tB := setupTwoNodeQuicPooled(t)
	t.Cleanup(func() { _ = nodeA.Close() })
	t.Cleanup(func() { _ = nodeB.Close() })
	t.Cleanup(func() { _ = tA.Close() })
	t.Cleanup(func() { _ = tB.Close() })

	return &quicCluster{
		G:     gomega.NewWithT(t),
		Ctx:   context.Background(),
		NodeA: nodeA,
		NodeB: nodeB,
		TA:    tA,
		TB:    tB,
	}
}

// TestQuicPooled_MapConvergence verifies that persistent BiStream pooling
// delivers ops correctly — the framing protocol doesn't corrupt data.
func TestQuicPooled_MapConvergence(t *testing.T) {
	c := newQuicClusterPooled(t)

	c.G.Expect(c.NodeA.(metaengine.MapBackend).MapSet(c.Ctx, "users", "u1",
		map[string]any{"name": "Alice"})).To(gomega.Succeed())

	eventuallyGet(c.G, c.NodeB, "users", "u1",
		map[string]any{"name": "Alice"}, 10*time.Second)
}

// TestQuicPooled_MultipleOpsSameStream verifies that multiple sequential ops
// are multiplexed over the same persistent BiStream without data corruption.
// Each op must arrive intact — framing boundaries must be respected.
func TestQuicPooled_MultipleOpsSameStream(t *testing.T) {
	c := newQuicClusterPooled(t)

	// Send 20 ops from A → B over the same pooled stream.
	for i := range 20 {
		key := fmt.Sprintf("key-%d", i)
		val := fmt.Sprintf("val-%d", i)
		c.G.Expect(c.NodeA.(metaengine.MapBackend).MapSet(c.Ctx, "kv", key, val)).
			To(gomega.Succeed())
	}

	// Verify all 20 arrived intact.
	for i := range 20 {
		key := fmt.Sprintf("key-%d", i)
		expected := fmt.Sprintf("val-%d", i)
		eventuallyGet(c.G, c.NodeB, "kv", key, expected, 10*time.Second)
	}
}

// TestQuicPooled_Bidirectional verifies ops flow in both directions over
// separate pooled streams (A→B and B→A each have their own persistent stream).
func TestQuicPooled_Bidirectional(t *testing.T) {
	c := newQuicClusterPooled(t)

	c.G.Expect(c.NodeA.(metaengine.MapBackend).MapSet(c.Ctx, "orders", "o1", "pending")).
		To(gomega.Succeed())
	eventuallyGet(c.G, c.NodeB, "orders", "o1", "pending", 10*time.Second)

	c.G.Expect(c.NodeB.(metaengine.MapBackend).MapSet(c.Ctx, "orders", "o2", "shipped")).
		To(gomega.Succeed())
	eventuallyGet(c.G, c.NodeA, "orders", "o2", "shipped", 10*time.Second)
}

// TestQuicPooled_StreamReuse verifies that multiple ops over a pooled connection
// reuse the same persistent BiStream. The stream-reuse counter on peerConn should
// show exactly 1 stream opened regardless of how many ops are sent.
func TestQuicPooled_StreamReuse(t *testing.T) {
	c := newQuicClusterPooled(t)

	// Send 20 ops from A → B over the same pooled stream.
	for i := range 20 {
		key := fmt.Sprintf("reuse-%d", i)
		val := fmt.Sprintf("val-%d", i)
		c.G.Expect(c.NodeA.(metaengine.MapBackend).MapSet(c.Ctx, "kv", key, val)).
			To(gomega.Succeed())
	}

	// Wait for at least some ops to arrive so the stream has been opened.
	eventuallyGet(c.G, c.NodeB, "kv", "reuse-0", "val-0", 10*time.Second)

	// Assert stream reuse: 20 ops should have used only 1 BiStream.
	streams := c.TA.StreamsOpenedForPeer(c.TB.NodeID())
	c.G.Expect(streams).To(gomega.Equal(int64(1)),
		"20 ops over pooled connection should reuse 1 stream, got %d", streams)
}

// TestQuicPooledToNonPooled_NoHang verifies that a pooled sender connected to a
// non-pooled receiver does NOT silently hang. The receiver detects the magic byte
// prefix and returns immediately instead of blocking in ReadToEnd waiting for a
// Finish() that never comes.
func TestQuicPooledToNonPooled_NoHang(t *testing.T) {
	g := gomega.NewWithT(t)

	// Pooled sender
	tA, err := quic.New(quic.WithLocalOnly(), quic.WithStreamPooling())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	t.Cleanup(func() { _ = tA.Close() })

	// Non-pooled receiver
	tB, err := quic.New(quic.WithLocalOnly())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	t.Cleanup(func() { _ = tB.Close() })

	nodeA := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-a"),
		irohengine.WithTransport(tA),
	)
	nodeB := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-b"),
		irohengine.WithTransport(tB),
	)
	t.Cleanup(func() { _ = nodeA.Close() })
	t.Cleanup(func() { _ = nodeB.Close() })

	ticketA, err := tA.Ticket()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(tB.Connect(ticketA)).To(gomega.Succeed())
	waitForPeers(t, []*quic.QuicTransport{tA, tB}, 1)

	// Publish from pooled sender. This must not hang — the non-pooled receiver
	// detects the protocol mismatch via the magic byte and returns immediately.
	done := make(chan struct{})
	go func() {
		_ = nodeA.(metaengine.MapBackend).MapSet(context.Background(), "mismatch", "k", "v")
		close(done)
	}()
	g.Eventually(done, 10*time.Second).Should(gomega.BeClosed(),
		"pooled sender to non-pooled receiver must not hang")

	// The op must NOT arrive at the non-pooled receiver (protocol mismatch detected).
	time.Sleep(500 * time.Millisecond)
	_, ok, _ := nodeB.(metaengine.MapBackend).MapGet(context.Background(), "mismatch", "k")
	g.Expect(ok).To(gomega.BeFalse(),
		"op should not arrive at non-pooled receiver after protocol mismatch")
}
