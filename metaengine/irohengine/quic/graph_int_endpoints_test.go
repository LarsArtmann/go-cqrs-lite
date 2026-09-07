//go:build cgo

package quic_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// quicGraphIntDispatch is the local interface for graph ops used by the
// int-endpoint test (write + read subset).
type quicGraphIntDispatch interface {
	GraphAddEdge(ctx context.Context, collection string, edge metaengine.Edge) error
	GraphRemoveEdge(ctx context.Context, collection string, edge metaengine.Edge) error
	GraphNeighbors(ctx context.Context, collection string, node any, depth int) ([]any, error)
}

// TestGraphIntEndpointsConvergeOverQuic pins non-string node endpoints
// across the QUIC transport. The QUIC decoder normalizes CBOR integers back
// to Go int (decodeOp's normalizeAny), so peers apply ops with int endpoints;
// memory-engine adjacency stringifies regardless. Together with the loopback
// twin (which receives uint64 endpoints un-normalized) this proves graph
// convergence is endpoint-type-independent on BOTH wire encodings.
func TestGraphIntEndpointsConvergeOverQuic(t *testing.T) {
	nodeA, nodeB, tA, tB := setupTwoNodeQuic(t)
	t.Cleanup(func() { _ = nodeA.Close() })
	t.Cleanup(func() { _ = nodeB.Close() })
	t.Cleanup(func() { _ = tA.Close() })
	t.Cleanup(func() { _ = tB.Close() })

	a := nodeA.(quicGraphIntDispatch)
	b := nodeB.(quicGraphIntDispatch)

	if err := a.GraphAddEdge(context.Background(), "hops", metaengine.Edge{From: 7, To: 8}); err != nil {
		t.Fatalf("GraphAddEdge: %v", err)
	}
	waitQuicIntNeighbors(t, b, "hops", 7, 1, []string{"8"},
		"edge with int endpoints must reach the peer")

	if err := b.GraphAddEdge(context.Background(), "hops", metaengine.Edge{From: 8, To: 9}); err != nil {
		t.Fatalf("GraphAddEdge: %v", err)
	}
	waitQuicIntNeighbors(t, a, "hops", 7, 2, []string{"8", "9"},
		"depth-2 int-endpoint traversal must converge")

	if err := a.GraphRemoveEdge(context.Background(), "hops", metaengine.Edge{From: 7, To: 8}); err != nil {
		t.Fatalf("GraphRemoveEdge: %v", err)
	}
	waitQuicIntNeighbors(t, b, "hops", 7, 1, nil,
		"removed int-endpoint edge must converge away")
}

// waitQuicIntNeighbors polls until node's neighbor set for the int endpoint
// matches expected exactly (stringified, order-insensitive).
func waitQuicIntNeighbors(
	t *testing.T,
	node quicGraphIntDispatch,
	collection string,
	start int,
	depth int,
	expected []string,
	msg string,
) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	var last []any

	for time.Now().Before(deadline) {
		neighbors, err := node.GraphNeighbors(context.Background(), collection, start, depth)
		if err == nil {
			last = neighbors
			if sameQuicNeighbors(neighbors, expected) {
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("%s: timeout waiting for neighbors of %d: got %v, want %v",
		msg, start, last, expected)
}

// sameQuicNeighbors compares a []any neighbor set against expected strings,
// order-insensitively.
func sameQuicNeighbors(actual []any, expected []string) bool {
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
