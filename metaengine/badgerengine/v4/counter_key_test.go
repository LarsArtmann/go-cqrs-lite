package badgerengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestBadger_CounterKeyComposite pins the composite-shaped counter-key
// contract (the tq row-683 class) on the Badger engine.
func TestBadger_CounterKeyComposite(t *testing.T) {
	t.Parallel()

	eng := mustNewBadgerEngine(t)

	enginetest.RunCounterKeyCompositeTest(t, eng)
}
