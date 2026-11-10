package bigtableengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestBigtable_CounterKeyComposite pins the composite-shaped counter-key
// contract (the tq row-683 class) on the Bigtable engine (in-process bttest
// fake).
func TestBigtable_CounterKeyComposite(t *testing.T) {
	t.Parallel()

	eng := newFakeEngine(t)

	enginetest.RunCounterKeyCompositeTest(t, eng)
}
