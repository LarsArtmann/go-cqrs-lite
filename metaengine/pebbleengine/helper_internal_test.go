package pebbleengine

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func mustNewPebbleEngineInternal(tb testing.TB) metaengine.Engine {
	tb.Helper()

	eng, err := NewPebbleEngine("")
	if err != nil {
		tb.Skipf("Pebble not available: %v", err)
	}

	tb.Cleanup(func() { _ = eng.Close() })

	return eng
}
