package metaengine_test

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestMemory_CounterKeyComposite pins the composite-shaped counter-key
// contract (the tq row-683 class) on the in-memory reference engine.
func TestMemory_CounterKeyComposite(t *testing.T) {
	t.Parallel()

	enginetest.RunCounterKeyCompositeTest(t, metaengine.NewMemoryEngine())
}
