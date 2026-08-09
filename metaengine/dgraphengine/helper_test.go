package dgraphengine_test

import (
	"os"
	"testing"

	dgraphengine "github.com/larsartmann/go-cqrs-lite/metaengine/dgraphengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// dgraphAddr returns the Dgraph gRPC address from DGRAPH_ADDR or defaults
// to localhost:9080.
func dgraphAddr() string {
	if addr := os.Getenv("DGRAPH_ADDR"); addr != "" {
		return addr
	}

	return "localhost:9080"
}

func mustNewDgraphEngine(tb testing.TB) metaengine.Engine {
	tb.Helper()

	eng, err := dgraphengine.New(dgraphAddr())
	if err != nil {
		tb.Skipf("Dgraph not available: %v", err)
	}

	tb.Cleanup(func() { _ = eng.Close() })

	return eng
}

func newDgraphEngineOrSkip(tb testing.TB) metaengine.Engine {
	tb.Helper()

	eng, err := dgraphengine.New(dgraphAddr())
	if err != nil {
		tb.Skipf("Dgraph not available: %v", err)
	}

	return eng
}
