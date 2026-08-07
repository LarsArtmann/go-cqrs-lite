package irohengine_test

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
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
