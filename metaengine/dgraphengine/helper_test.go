package dgraphengine_test

import (
	"fmt"
	"os"
	"strings"
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
	return fmt.Sprintf("%s_%x_%d", base, os.Getpid(), collSeq.Add(1))
}

var collSeq atomic.Uint64 //nolint:gochecknoglobals // test-only unique suffix source

// dgraphAddr returns the Dgraph gRPC address from DGRAPH_ADDR or defaults
// to localhost:9080.
func dgraphAddr() string {
	if addr := os.Getenv("DGRAPH_ADDR"); addr != "" {
		return addr
	}

	return "localhost:9080"
}

// dgraphSkipClass reports whether err is the server-not-reachable class
// (OQ #10 skip-vs-fail policy): a Dgraph server that is not running, not yet
// listening, or unreachable makes live coverage impossible without a defect,
// so the test skips. EVERYTHING ELSE — contention exhausted after retries,
// auth failures, unexpected server errors — fails loudly: those are exactly
// the classes that once silently deleted four ADT subtests from the suite
// (2026-09-11, gotchas-testing.md).
func dgraphSkipClass(err error) bool {
	msg := strings.ToLower(err.Error())

	for _, marker := range []string{
		"connection refused",
		"error while dialing",
		"no such host",
		"connection timed out",
		"name resolver",
		"transport is closing",
		"code = unavailable", // gRPC connectivity, not a Dgraph error
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}

	return false
}

func mustNewDgraphEngine(tb testing.TB) metaengine.Engine {
	tb.Helper()

	eng, err := dgraphengine.New(dgraphAddr())
	if err != nil {
		if dgraphSkipClass(err) {
			tb.Skipf("Dgraph not available: %v", err)
		}

		tb.Fatalf("dgraph engine construction failed (not a skip-class error): %v", err)
	}

	tb.Cleanup(func() { _ = eng.Close() })

	return eng
}

func newDgraphEngineOrSkip(tb testing.TB) metaengine.Engine {
	tb.Helper()

	eng, err := dgraphengine.New(dgraphAddr())
	if err != nil {
		if dgraphSkipClass(err) {
			tb.Skipf("Dgraph not available: %v", err)
		}

		tb.Fatalf("dgraph engine construction failed (not a skip-class error): %v", err)
	}

	return eng
}
