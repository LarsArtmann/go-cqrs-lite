package bboltengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestBbolt_CounterKeyComposite pins the composite-shaped counter-key
// contract (the tq row-683 class) on the bbolt engine.
func TestBbolt_CounterKeyComposite(t *testing.T) {
	t.Parallel()

	eng := mustNewBboltEngine(t)

	enginetest.RunCounterKeyCompositeTest(t, eng)
}
