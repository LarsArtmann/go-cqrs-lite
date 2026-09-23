package metaengine

import (
	"context"
	"fmt"
)

// GraphBFS walks a graph breadth-first from node up to depth levels,
// expanding each frontier node with expand (directed: one neighbor query;
// undirected: outgoing+incoming). encode turns the caller's node value into
// the adjacency key; label prefixes traversal errors. The visited set
// deduplicates across levels; the result is never nil. SQL engines whose
// servers lack WITH RECURSIVE use this as their iterative fallback.
func GraphBFS(
	ctx context.Context,
	node any,
	depth int,
	label string,
	encode func(any) string,
	expand func(ctx context.Context, n string) ([]string, error),
) ([]any, error) {
	startNode := encode(node)
	visited := map[string]bool{startNode: true}
	frontier := []string{startNode}
	var result []any

	for level := 0; level < depth && len(frontier) > 0; level++ {
		var next []string

		for _, n := range frontier {
			neighbors, err := expand(ctx, n)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", label, err)
			}

			for _, nb := range neighbors {
				if visited[nb] {
					continue
				}

				visited[nb] = true
				result = append(result, nb)
				next = append(next, nb)
			}
		}

		frontier = next
	}

	if result == nil {
		result = []any{}
	}

	return result, nil
}
