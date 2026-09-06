package irohengine

import (
	"context"
	"reflect"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Polling helpers for the convergence suite (convergence_suite.go) — they work
// with both synchronous (in-process) and asynchronous (TCP/QUIC) transports by
// polling until replicas agree or the deadline expires.

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

// graphDispatch mirrors metaengine's unexported graph dispatch contract
// (ADR-0113) for the convergence suite's graph scenarios.
type graphDispatch interface {
	GraphAddEdge(ctx context.Context, collection string, edge metaengine.Edge) error
	GraphNeighbors(ctx context.Context, collection string, node any, depth int) ([]any, error)
}

// graphRemoveDispatch mirrors the optional edge-removal extension.
type graphRemoveDispatch interface {
	GraphRemoveEdge(ctx context.Context, collection string, edge metaengine.Edge) error
}

// waitForGraphNeighbors polls until the depth-limited neighbor set of node
// matches expected exactly (order-insensitive). An empty expected asserts the
// node has no neighbors — the edge-removal convergence shape.
func waitForGraphNeighbors(
	t *testing.T,
	node metaengine.Engine,
	collection, start string,
	depth int,
	expected []string,
) {
	t.Helper()
	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		neighbors, err := node.(graphDispatch).GraphNeighbors(
			context.Background(), collection, start, depth)
		if err == nil && sameSetAny(neighbors, expected) {
			return
		}
		time.Sleep(pollInterval)
	}
	neighbors, _ := node.(graphDispatch).GraphNeighbors(
		context.Background(), collection, start, depth)
	t.Fatalf("timeout: %s neighbors of %s (depth %d) expected %v (got %v)",
		collection, start, depth, expected, neighbors)
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
