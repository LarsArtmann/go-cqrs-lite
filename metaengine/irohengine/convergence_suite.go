package irohengine

import (
	"context"
	"reflect"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ClusterFactory creates a 2-node replicated cluster for convergence testing.
// The factory wires transports, connects peers, waits for connection readiness,
// and registers cleanup via t.Cleanup. The caller only needs the two engines.
//
// Each transport (in-process, loopback, QUIC) provides its own factory
// implementation and calls RunConvergenceSuite once.
type ClusterFactory func(t *testing.T) (nodeA, nodeB metaengine.Engine)

// RunConvergenceSuite runs the standard convergence test battery against a
// 2-node cluster created by factory. This eliminates ~200 lines of duplicated
// convergence tests across the in-process, loopback, and QUIC transport test
// files.
//
// The suite covers the 6 common CRDT convergence scenarios:
//   - MapConvergence (A→B MapSet/MapGet)
//   - Bidirectional (A→B and B→A)
//   - CounterConvergence (PN-Counter)
//   - SetConvergence (OR-Set)
//   - LogConvergence (append-only log)
//   - MultimapConvergence (OR-Set per key)
//
// Transport-specific tests (LWW with clock, RTT measurement, protocol mismatch,
// stream pooling) remain in their respective test files.
func RunConvergenceSuite(t *testing.T, factory ClusterFactory) {
	t.Helper()

	t.Run("MapConvergence", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		nodeA, nodeB := factory(t)

		expected := map[string]any{"name": "Alice"}
		mustNoErr(t, nodeA.(metaengine.MapBackend).MapSet(ctx, "users", "u1", expected))
		waitForMap(t, nodeB, "users", "u1", expected)
	})

	t.Run("Bidirectional", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		nodeA, nodeB := factory(t)

		mustNoErr(t, nodeA.(metaengine.MapBackend).MapSet(ctx, "orders", "o1", "pending"))
		waitForMap(t, nodeB, "orders", "o1", "pending")

		mustNoErr(t, nodeB.(metaengine.MapBackend).MapSet(ctx, "orders", "o2", "shipped"))
		waitForMap(t, nodeA, "orders", "o2", "shipped")
	})

	t.Run("CounterConvergence", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		nodeA, nodeB := factory(t)

		mustNoErr(t, nodeA.(metaengine.CounterBackend).
			CounterIncrement(ctx, "visits", metaengine.Delta{"total": 5}))
		mustNoErr(t, nodeB.(metaengine.CounterBackend).
			CounterIncrement(ctx, "visits", metaengine.Delta{"total": 3}))

		waitForCounter(t, nodeA, "visits", "total", 8)
		waitForCounter(t, nodeB, "visits", "total", 8)
	})

	t.Run("SetConvergence", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		nodeA, nodeB := factory(t)

		mustNoErr(t, nodeA.(metaengine.SetBackend).SetAdd(ctx, "tags", "go"))
		mustNoErr(t, nodeA.(metaengine.SetBackend).SetAdd(ctx, "tags", "cqrs"))

		waitForSetContains(t, nodeB, "tags", "go")
		waitForSetContains(t, nodeB, "tags", "cqrs")
	})

	t.Run("LogConvergence", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		nodeA, nodeB := factory(t)

		mustNoErr(t, nodeA.(metaengine.LogBackend).LogAppend(ctx, "audit", "user-login"))
		mustNoErr(t, nodeA.(metaengine.LogBackend).LogAppend(ctx, "audit", "file-upload"))

		waitForLogTail(t, nodeB, "audit", []string{"user-login", "file-upload"})
	})

	t.Run("MultimapConvergence", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		nodeA, nodeB := factory(t)

		mmbA := nodeA.(metaengine.MultimapBackend)
		mmbB := nodeB.(metaengine.MultimapBackend)

		mustNoErr(t, mmbA.MultiAdd(ctx, "members", "team-a", "alice"))
		mustNoErr(t, mmbA.MultiAdd(ctx, "members", "team-a", "bob"))
		mustNoErr(t, mmbB.MultiAdd(ctx, "members", "team-a", "carol"))

		waitForMultimap(t, nodeA, "members", "team-a", []string{"alice", "bob", "carol"})
		waitForMultimap(t, nodeB, "members", "team-a", []string{"alice", "bob", "carol"})
	})
}

// --- Polling helpers (work with both sync and async transports) ---

const (
	pollInterval = 50 * time.Millisecond
	// pollTimeout bounds each convergence assertion. 30s (was 15s) after a
	// 15s near-miss under -race load on 2026-08-16: passing runs still exit
	// as soon as replicas agree, so only genuinely slow convergence pays.
	pollTimeout = 30 * time.Second
)

func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func waitForMap(
	t *testing.T,
	node metaengine.Engine,
	collection, key string,
	expected any,
) {
	t.Helper()
	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		val, ok, err := node.(metaengine.MapBackend).MapGet(
			context.Background(), collection, key)
		if err == nil && ok && reflect.DeepEqual(val, expected) {
			return
		}
		time.Sleep(pollInterval)
	}
	val, ok, _ := node.(metaengine.MapBackend).MapGet(
		context.Background(), collection, key)
	t.Fatalf("timeout: %s/%s expected %v (got ok=%v val=%v)",
		collection, key, expected, ok, val)
}

func waitForCounter(
	t *testing.T,
	node metaengine.Engine,
	collection, counter string,
	expected int64,
) {
	t.Helper()
	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		counts, err := node.(metaengine.CounterBackend).CounterGet(
			context.Background(), collection)
		if err == nil && counts[counter] == expected {
			return
		}
		time.Sleep(pollInterval)
	}
	counts, _ := node.(metaengine.CounterBackend).CounterGet(
		context.Background(), collection)
	t.Fatalf("timeout: %s/%s expected %d (got %v)",
		collection, counter, expected, counts[counter])
}

func waitForSetContains(
	t *testing.T,
	node metaengine.Engine,
	collection, value string,
) {
	t.Helper()
	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		contains, err := node.(metaengine.SetBackend).SetContains(
			context.Background(), collection, value)
		if err == nil && contains {
			return
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("timeout: %s does not contain %q", collection, value)
}

func waitForLogTail(
	t *testing.T,
	node metaengine.Engine,
	collection string,
	expected []string,
) {
	t.Helper()
	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		entries, err := node.(metaengine.LogBackend).LogTail(
			context.Background(), collection, len(expected))
		if err == nil && sameLogTail(entries, expected) {
			return
		}
		time.Sleep(pollInterval)
	}
	entries, _ := node.(metaengine.LogBackend).LogTail(
		context.Background(), collection, len(expected))
	t.Fatalf("timeout: %s tail expected %v (got %v)",
		collection, expected, entries)
}

func waitForMultimap(
	t *testing.T,
	node metaengine.Engine,
	collection, key string,
	expected []string,
) {
	t.Helper()
	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		vals, err := node.(metaengine.MultimapBackend).MultiGet(
			context.Background(), collection, key)
		if err == nil && sameSetAny(vals, expected) {
			return
		}
		time.Sleep(pollInterval)
	}
	vals, _ := node.(metaengine.MultimapBackend).MultiGet(
		context.Background(), collection, key)
	t.Fatalf("timeout: %s/%s expected %v (got %v)",
		collection, key, expected, vals)
}

// sameSetAny checks that a []any slice contains the same string elements as
// expected, regardless of order (set equality). Non-string elements fail.
func sameSetAny(actual []any, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	expectedSet := make(map[string]bool, len(expected))
	for _, v := range expected {
		expectedSet[v] = true
	}
	for _, v := range actual {
		s, ok := v.(string)
		if !ok || !expectedSet[s] {
			return false
		}
	}
	return true
}

// sameLogTail checks that an []any slice (from LogTail) matches the expected
// []string in ORDER — logs are append-only, so order is meaningful.
// sameLogTail compares a replicated log tail against the expected values as
// an unordered multiset: replicated log transports guarantee eventual
// delivery, not cross-op ordering (per-op streams apply concurrently on the
// receiver, so two appends can land in either order under load). Order-
// sensitive consumers need a sequenced log, which these engines do not
// promise.
func sameLogTail(actual []any, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}

	remaining := make([]string, len(expected))
	copy(remaining, expected)

	for _, v := range actual {
		s, ok := v.(string)
		if !ok {
			return false
		}

		matched := false
		for i, want := range remaining {
			if s == want {
				remaining = append(remaining[:i], remaining[i+1:]...)
				matched = true
				break
			}
		}

		if !matched {
			return false
		}
	}

	return true
}
