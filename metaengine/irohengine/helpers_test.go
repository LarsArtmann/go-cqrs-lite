package irohengine_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// newTwoNodeCluster is the shared fixture for convergence tests: a 2-node
// replicated cluster (each node a Memory engine) wired through an in-process
// network. Cleanup is registered with t.Cleanup, so callers need no defers.
func newTwoNodeCluster(t *testing.T) (nodeA, nodeB metaengine.Engine) {
	t.Helper()
	net := irohengine.NewNetwork()
	nodeA = irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-a"),
		irohengine.WithTransport(net.Join("a")),
	)
	nodeB = irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-b"),
		irohengine.WithTransport(net.Join("b")),
	)
	t.Cleanup(func() {
		_ = nodeA.Close()
		_ = nodeB.Close()
	})
	return nodeA, nodeB
}

// manualClock is a deterministic Clock for tests. It starts at a fixed epoch
// and only advances when Advance is called — eliminating all timing assumptions.
type manualClock struct {
	now atomic.Int64 // unix-nanos
}

func newManualClock(start time.Time) *manualClock {
	c := &manualClock{}
	c.now.Store(start.UnixNano())
	return c
}

func (c *manualClock) Now() time.Time {
	return time.Unix(0, c.now.Load())
}

// Advance moves the clock forward by d and returns the new time.
func (c *manualClock) Advance(d time.Duration) time.Time {
	newNanos := c.now.Add(int64(d))
	return time.Unix(0, newNanos)
}

// newTwoNodeClusterWithClock creates a cluster where both nodes share the same
// injectable clock, enabling deterministic LWW timestamp ordering without
// time.Sleep. The clock starts at 2026-01-01T00:00:00Z.
func newTwoNodeClusterWithClock(
	t *testing.T,
	clock *manualClock,
) (nodeA, nodeB metaengine.Engine) {
	t.Helper()
	net := irohengine.NewNetwork()
	nodeA = irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-a"),
		irohengine.WithTransport(net.Join("a")),
		irohengine.WithClock(clock),
	)
	nodeB = irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node-b"),
		irohengine.WithTransport(net.Join("b")),
		irohengine.WithClock(clock),
	)
	t.Cleanup(func() {
		_ = nodeA.Close()
		_ = nodeB.Close()
	})
	return nodeA, nodeB
}
