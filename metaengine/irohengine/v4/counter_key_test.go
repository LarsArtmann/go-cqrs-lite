package irohengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestIroh_CounterKeyComposite pins the composite-shaped counter-key
// contract (the tq row-683 class) on a single replicated node (memory store
// behind the iroh transport).
func TestIroh_CounterKeyComposite(t *testing.T) {
	t.Parallel()

	net := irohengine.NewNetwork()
	node := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("counterkey"),
		irohengine.WithTransport(net.Join("a")),
	)
	t.Cleanup(func() { _ = node.Close() })

	enginetest.RunCounterKeyCompositeTest(t, node)
}
