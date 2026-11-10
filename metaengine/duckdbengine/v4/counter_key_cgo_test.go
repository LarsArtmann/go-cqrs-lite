//go:build cgo

package duckdbengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestDuckDB_CounterKeyComposite pins the composite-shaped counter-key
// contract (the tq row-683 class) on the DuckDB engine.
func TestDuckDB_CounterKeyComposite(t *testing.T) {
	t.Parallel()

	eng := mustNewDuckEngine(t)

	enginetest.RunCounterKeyCompositeTest(t, eng)
}
