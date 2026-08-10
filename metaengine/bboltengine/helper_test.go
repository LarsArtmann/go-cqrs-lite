package bboltengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func mustNewBboltEngine(tb testing.TB) metaengine.Engine {
	tb.Helper()

	eng, err := bboltengine.NewBboltEngine("")
	if err != nil {
		tb.Skipf("bbolt not available: %v", err)
	}

	tb.Cleanup(func() { _ = eng.Close() })

	return eng
}

func newBboltEngineOrSkip(tb testing.TB) metaengine.Engine {
	tb.Helper()

	eng, err := bboltengine.NewBboltEngine("")
	if err != nil {
		tb.Skipf("bbolt not available: %v", err)
	}

	return eng
}
