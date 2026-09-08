package loopback_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// graphIntDispatch is the local interface for graph ops used by the
// int-endpoint tests (write + read subset).
type graphIntDispatch interface {
	GraphAddEdge(ctx context.Context, collection string, edge metaengine.Edge) error
	GraphRemoveEdge(ctx context.Context, collection string, edge metaengine.Edge) error
	GraphNeighbors(ctx context.Context, collection string, node any, depth int) ([]any, error)
}

// TestGraphIntEndpointsConvergeOverTCP pins non-string node endpoints across
// the TCP transport. CBOR encodes Go int as unsigned and the loopback decoder
// has no int-normalization, so the peer's ops carry uint64 endpoints; graph
// convergence must NOT depend on that (memory-engine adjacency is stringified,
// and the per-edge LWW key stringifies too). A regression that stored raw
// `any` endpoints or mixed typed LWW keys would make edges visible on one
// node but invisible on the peer — exactly what this test fails on.
func TestGraphIntEndpointsConvergeOverTCP(t *testing.T) {
	t.Parallel()

	nodeA, nodeB, tA, tB := setupTwoNodeLoopback(t)
	t.Cleanup(func() { _ = nodeA.Close() })
	t.Cleanup(func() { _ = nodeB.Close() })
	t.Cleanup(func() { _ = tA.Close() })
	t.Cleanup(func() { _ = tB.Close() })

	a := nodeA.(graphIntDispatch)
	b := nodeB.(graphIntDispatch)

	gAdd(t, a, "hops", 7, 8)
	waitForIntNeighbors(t, b, "hops", 7, 1, []string{"8"},
		"edge with int endpoints must reach the peer")

	// Depth-2 chain written from the PEER side (B→A direction) with int
	// endpoints: 8→9 must be visible from 7 on A via traversal.
	gAdd(t, b, "hops", 8, 9)
	waitForIntNeighbors(t, a, "hops", 7, 2, []string{"8", "9"},
		"depth-2 int-endpoint traversal must converge")

	// Removal with int endpoints must converge away on the peer.
	gRemove(t, a, "hops", 7, 8)
	waitForIntNeighbors(t, b, "hops", 7, 1, nil,
		"removed int-endpoint edge must converge away")
}

func gAdd(t *testing.T, e graphIntDispatch, col string, from, to int) {
	t.Helper()
	if err := e.GraphAddEdge(
		context.Background(),
		col,
		metaengine.Edge{From: from, To: to},
	); err != nil {
		t.Fatalf("GraphAddEdge %d→%d: %v", from, to, err)
	}
}

func gRemove(t *testing.T, e graphIntDispatch, col string, from, to int) {
	t.Helper()
	if err := e.GraphRemoveEdge(
		context.Background(),
		col,
		metaengine.Edge{From: from, To: to},
	); err != nil {
		t.Fatalf("GraphRemoveEdge %d→%d: %v", from, to, err)
	}
}

// waitForIntNeighbors polls until node's neighbor set for the int endpoint
// matches expected exactly (stringified, order-insensitive). Loopback
// delivery is async — the write returns before the peer applies.
func waitForIntNeighbors(
	t *testing.T,
	node graphIntDispatch,
	collection string,
	start int,
	depth int,
	expected []string,
	msg string,
) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	var last []any

	for time.Now().Before(deadline) {
		neighbors, err := node.GraphNeighbors(context.Background(), collection, start, depth)
		if err == nil {
			last = neighbors
			if sameNeighbors(neighbors, expected) {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("%s: timeout waiting for neighbors of %d: got %v, want %v",
		msg, start, last, expected)
}

// sameNeighbors compares a []any neighbor set against expected strings,
// order-insensitively.
func sameNeighbors(actual []any, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}

	seen := make(map[string]bool, len(actual))
	for _, v := range actual {
		seen[fmt.Sprint(v)] = true
	}

	for _, v := range expected {
		if !seen[v] {
			return false
		}
	}

	return true
}
