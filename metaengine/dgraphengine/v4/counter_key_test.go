package dgraphengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestDgraph_CounterKeyComposite pins the composite-shaped counter-key
// contract (the tq row-683 class) on the Dgraph engine. Skips when no
// Dgraph instance is available (DGRAPH_ADDR env or localhost:9080 default).
func TestDgraph_CounterKeyComposite(t *testing.T) {
	t.Parallel()

	eng := mustNewDgraphEngine(t)

	enginetest.RunCounterKeyCompositeTest(t, eng)
}
