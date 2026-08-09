package badgerengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func mustNewBadgerEngine(tb testing.TB) metaengine.Engine {
	tb.Helper()

	eng, err := badgerengine.NewBadgerEngine("")
	if err != nil {
		tb.Skipf("Badger not available: %v", err)
	}

	tb.Cleanup(func() { _ = eng.Close() })

	return eng
}

func newBadgerEngineOrSkip(tb testing.TB) metaengine.Engine {
	tb.Helper()

	eng, err := badgerengine.NewBadgerEngine("")
	if err != nil {
		tb.Skipf("Badger not available: %v", err)
	}

	return eng
}
