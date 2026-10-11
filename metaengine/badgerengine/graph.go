package badgerengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"strconv"
	"strings"

	"github.com/dgraph-io/badger/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/keycodec"
)

// --- graphBackend (native, prefix-scan) ---
//
// Edges are stored twice as marker keys: forward adjacency under
// "edge\x00<col>\x00<from>\x00<to>" and reverse adjacency under
// "edger\x00<col>\x00<to>\x00<from>". An LSM prefix seek makes each
// expansion step O(degree) — the same class as the SQL engines' indexed
// meta_graph_edges lookups, and far better than the degraded multimap BFS
// fallback (O(N * degree^depth)). The reverse index gives undirected
// traversal and keeps GraphRemoveEdge a two-key delete.

// graphNodeKey renders a node as the canonical adjacency key form. Mirrors
// sqlite/pg encodeNodeKey so cross-engine node representations agree.
// art-dupl:accept cross-module engine pattern — separate go.mod
func graphNodeKey(key any) string {
	switch k := key.(type) {
	case string:
		return k
	case int:
		return strconv.Itoa(k)
	case int64:
		return strconv.FormatInt(k, 10)
	case int32:
		return strconv.FormatInt(int64(k), 10)
	case uint64:
		return strconv.FormatUint(k, 10)
	case uint32:
		return strconv.FormatUint(uint64(k), 10)
	default:
		return graphNodeKeyJSON(key)
	}
}

func graphNodeKeyJSON(v any) string {
	// art-dupl:accept marshal-with-fallback helper; sqliteengine encodeJSON twin is dep-isolated
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}

	return string(b)
}

// GraphAddEdge stores both adjacency directions. Idempotent: re-adding an
// existing edge overwrites the same marker keys.
func (e *badgerEngine) GraphAddEdge(
	_ context.Context,
	collection string,
	edge metaengine.Edge,
) error {
	// art-dupl:accept add/remove edge share the key-encoding prologue by symmetry
	from := graphNodeKey(edge.From)
	to := graphNodeKey(edge.To)

	fwd := keycodec.GraphEdgeFwdKey(collection, from, to)
	rev := keycodec.GraphEdgeRevKey(collection, to, from)

	return e.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(fwd, []byte{}); err != nil {
			return fmt.Errorf("badgerengine.GraphAddEdge: %w", err)
		}

		if err := txn.Set(rev, []byte{}); err != nil {
			return fmt.Errorf("badgerengine.GraphAddEdge: %w", err)
		}

		return nil
	})
}

// GraphRemoveEdge deletes both adjacency markers (ADR-0114 style tombstone
// dispatch). Idempotent: removing a missing edge is a no-op.
func (e *badgerEngine) GraphRemoveEdge(
	_ context.Context,
	collection string,
	edge metaengine.Edge,
) error {
	from := graphNodeKey(edge.From)
	to := graphNodeKey(edge.To)

	fwd := keycodec.GraphEdgeFwdKey(collection, from, to)
	rev := keycodec.GraphEdgeRevKey(collection, to, from)

	return e.db.Update(func(txn *badger.Txn) error {
		if err := txn.Delete(fwd); err != nil {
			return fmt.Errorf("badgerengine.GraphRemoveEdge: %w", err)
		}

		if err := txn.Delete(rev); err != nil {
			return fmt.Errorf("badgerengine.GraphRemoveEdge: %w", err)
		}

		return nil
	})
}

// GraphNeighbors returns all nodes reachable from node within depth hops
// (directed), via BFS with one prefix scan per visited node per level.
func (e *badgerEngine) GraphNeighbors(
	_ context.Context,
	collection string,
	node any,
	depth int,
) ([]any, error) {
	// art-dupl:accept directed/undirected BFS share guard+init prologue
	if depth <= 0 {
		return []any{}, nil
	}

	start := graphNodeKey(node)
	visited := map[string]bool{start: true}
	frontier := []string{start}
	var result []any

	err := e.db.View(func(txn *badger.Txn) error {
		for level := 0; level < depth && len(frontier) > 0; level++ {
			var next []string

			for _, n := range frontier {
				neighbors, err := scanEdgeSuffixes(txn, keycodec.GraphEdgeFwdPrefix(collection, n))
				if err != nil {
					return fmt.Errorf("badgerengine.GraphNeighbors: %w", err)
				}

				for _, nb := range neighbors {
					if !visited[nb] {
						visited[nb] = true
						result = append(result, nb)
						next = append(next, nb)
					}
				}
			}

			frontier = next
		}

		return nil
	})
	// art-dupl:accept view-error epilogue shared by both neighbor walks
	if err != nil {
		return nil, err //nolint:wrapcheck // already prefixed inside the view
	}

	if result == nil {
		result = []any{}
	}

	return result, nil
}

// GraphNeighborsUndirected walks edges in BOTH directions: each expansion
// merges the forward prefix scan with the reverse-adjacency prefix scan.
func (e *badgerEngine) GraphNeighborsUndirected(
	_ context.Context,
	collection string,
	node any,
	depth int,
) ([]any, error) {
	if depth <= 0 {
		return []any{}, nil
	}

	start := graphNodeKey(node)
	visited := map[string]bool{start: true}
	frontier := []string{start}
	var result []any

	err := e.db.View(func(txn *badger.Txn) error {
		for level := 0; level < depth && len(frontier) > 0; level++ {
			var next []string

			for _, n := range frontier {
				outgoing, err := scanEdgeSuffixes(txn, keycodec.GraphEdgeFwdPrefix(collection, n))
				if err != nil {
					return fmt.Errorf("badgerengine.GraphNeighborsUndirected: %w", err)
				}

				incoming, err := scanEdgeSuffixes(txn, keycodec.GraphEdgeRevPrefix(collection, n))
				if err != nil {
					return fmt.Errorf("badgerengine.GraphNeighborsUndirected: %w", err)
				}

				for _, nb := range append(outgoing, incoming...) {
					if !visited[nb] {
						visited[nb] = true
						result = append(result, nb)
						next = append(next, nb)
					}
				}
			}

			frontier = next
		}

		return nil
	})
	// art-dupl:accept view-error epilogue shared by both neighbor walks
	if err != nil {
		return nil, err //nolint:wrapcheck // already prefixed inside the view
	}

	if result == nil {
		result = []any{}
	}

	return result, nil
}

// scanEdgeSuffixes prefix-scans an adjacency family and returns the trailing
// node segment of every matched key ("edge\x00col\x00from\x00TO" → "TO").
func scanEdgeSuffixes(txn *badger.Txn, prefix []byte) ([]string, error) {
	iter := txn.NewIterator(badger.IteratorOptions{Prefix: prefix})
	defer iter.Close()

	var nodes []string

	for iter.Seek(prefix); iter.ValidForPrefix(prefix); iter.Next() {
		nodes = append(nodes, strings.TrimPrefix(string(iter.Item().Key()), string(prefix)))
	}

	return nodes, nil
}
