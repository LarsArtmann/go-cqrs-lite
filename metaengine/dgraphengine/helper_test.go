package dgraphengine_test

import (
	"fmt"
	"os"
	"sync/atomic"
	"testing"

	dgraphengine "github.com/larsartmann/go-cqrs-lite/metaengine/dgraphengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// uniqueCollection returns base with a per-run unique suffix (pid + in-process
// counter). Tests running against a SHARED PERSISTENT Dgraph server would
// otherwise collide with leftovers from previous runs or from `-count>1`
// re-executions within one process (fixed collection names like "products"
// accumulate stale nodes and poison count/scan assertions). Unique names make
// every run idempotent without destructive drops.
func uniqueCollection(tb testing.TB, base string) string {
	tb.Helper()
	return fmt.Sprintf("%s_%x_%d", base, os.Getpid(), atomic.AddUint64(&collSeq, 1))
}

var collSeq uint64 //nolint:gochecknoglobals // test-only unique suffix source

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
