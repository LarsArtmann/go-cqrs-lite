package pebbleengine_test

import (
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
