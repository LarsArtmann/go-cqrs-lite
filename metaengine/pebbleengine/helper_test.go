package pebbleengine_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func mustNewPebbleEngine(tb testing.TB) metaengine.Engine {
	tb.Helper()

	eng, err := pebbleengine.NewPebbleEngine("")
	if err != nil {
		tb.Skipf("Pebble not available: %v", err)
	}

	tb.Cleanup(func() { _ = eng.Close() })

	return eng
}

func newPebbleEngineOrSkip(tb testing.TB) metaengine.Engine {
	tb.Helper()

	eng, err := pebbleengine.NewPebbleEngine("")
	if err != nil {
		tb.Skipf("Pebble not available: %v", err)
	}

	return eng
}

// layoutFixture bundles the capabilities the layout-planner tests exercise:
// the engine plus its LayoutPlanner and MapBackend facets.
type layoutFixture struct {
	ctx context.Context
	eng metaengine.Engine
	lp  metaengine.LayoutPlanner
	mb  metaengine.MapBackend
}

// newLayoutFixture creates a fresh Pebble engine and applies a layout for
// the given collection (filter fields, then sort fields).
func newLayoutFixture(tb testing.TB, col string, filterFields, sortFields []string) layoutFixture {
	tb.Helper()

	f := layoutFixture{
		ctx: context.Background(),
		eng: mustNewPebbleEngine(tb),
	}

	lp, ok := f.eng.(metaengine.LayoutPlanner)
	if !ok {
		tb.Fatal("expected pebbleEngine to implement LayoutPlanner")
	}

	f.lp = lp

	if err := lp.ApplyLayout(col, filterFields, sortFields); err != nil {
		tb.Fatalf("ApplyLayout: %v", err)
	}

	f.mb = f.eng.(metaengine.MapBackend)

	return f
}
