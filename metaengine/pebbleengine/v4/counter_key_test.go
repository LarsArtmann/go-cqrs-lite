package pebbleengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestPebble_CounterKeyComposite pins the composite-shaped counter-key
// contract (the tq row-683 class) on the Pebble engine.
func TestPebble_CounterKeyComposite(t *testing.T) {
	t.Parallel()

	eng := mustNewPebbleEngine(t)

	enginetest.RunCounterKeyCompositeTest(t, eng)
}
