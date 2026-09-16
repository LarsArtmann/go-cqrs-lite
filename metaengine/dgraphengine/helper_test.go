package dgraphengine_test

import (
	"fmt"
	"os"
	"strings"
	"sync"
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

// liveServerMu serializes destructive resets against every other live test:
// the suite shares ONE ephemeral Dgraph server, and ResetEngine is a TOTAL
// wipe (every engine node's predicates are nulled), so a reset running while
// parallel tests read or write silently deletes their data mid-test
// (observed 2026-09-16: GraphRAG searches returned 0 hits and ADT graph
// matrices saw empty results inside the TestResetEngine_Idempotent window).
// Reader tests hold RLock via the engine helpers for their whole body; the
// two parallel reset tests hold Lock via mustNewDgraphEngineExclusive. Go's
// RWMutex blocks new RLocks once a writer waits, so in-flight tests drain
// before the reset runs.
var liveServerMu sync.RWMutex

// newDgraphEngineWithLock is the shared construction path. The lock is
// registered for release via tb.Cleanup BEFORE construction so Skip/Fatal
// paths (which run cleanups via Goexit) still release it.
func newDgraphEngineWithLock(tb testing.TB, lock, unlock func()) metaengine.Engine {
	tb.Helper()

	lock()
	tb.Cleanup(unlock)

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

func mustNewDgraphEngine(tb testing.TB) metaengine.Engine {
	tb.Helper()

	return newDgraphEngineWithLock(tb, liveServerMu.RLock, liveServerMu.RUnlock)
}

func newDgraphEngineOrSkip(tb testing.TB) metaengine.Engine {
	tb.Helper()

	return newDgraphEngineWithLock(tb, liveServerMu.RLock, liveServerMu.RUnlock)
}

// mustNewDgraphEngineExclusive takes the server EXCLUSIVELY for the test's
// whole body. Required for every test that calls ResetEngine: the wipe must
// not race any parallel reader/writer, and its post-reset emptiness
// assertions must not observe foreign data. Do NOT call this while also
// holding the reader lock (same-goroutine RLock then Lock deadlocks).
func mustNewDgraphEngineExclusive(tb testing.TB) metaengine.Engine {
	tb.Helper()

	return newDgraphEngineWithLock(tb, liveServerMu.Lock, liveServerMu.Unlock)
}
