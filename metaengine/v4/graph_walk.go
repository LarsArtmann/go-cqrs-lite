package metaengine

import (
	"context"
	"fmt"
)

// GraphBFSNodes is the generic iterative breadth-first walk core: from node,
// up to depth levels, expanding each frontier element with expand. key turns
// a node into its dedup identity (SQL engines: the encoded adjacency key;
// value-faithful engines: a typed key so int(1) and "1" stay distinct nodes).
// label prefixes traversal errors. The visited set deduplicates across
// levels; the result is never nil (the nil-vs-empty contract, 2026-10-01).
func GraphBFSNodes[N any](
	ctx context.Context,
	node N,
	depth int,
	label string,
	key func(N) string,
	expand func(ctx context.Context, n N) ([]N, error),
) ([]N, error) {
	visited := map[string]bool{key(node): true}
	frontier := []N{node}
	var result []N

	for level := 0; level < depth && len(frontier) > 0; level++ {
		var next []N

		for _, n := range frontier {
			neighbors, err := expand(ctx, n)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", label, err)
			}

			for _, nb := range neighbors {
				k := key(nb)
				if visited[k] {
					continue
				}

				visited[k] = true
				result = append(result, nb)
				next = append(next, nb)
			}
		}

		frontier = next
	}

	if result == nil {
		result = []N{}
	}

	return result, nil
}

// GraphBFS is the string-keyed form of GraphBFSNodes for SQL engines whose
// adjacency space is stringly (one neighbor query per frontier node). encode
// turns the caller's node value into the adjacency key. The result is never
// nil. SQL engines whose servers lack WITH RECURSIVE use this as their
// iterative fallback.
func GraphBFS(
	ctx context.Context,
	node any,
	depth int,
	label string,
	encode func(any) string,
	expand func(ctx context.Context, n string) ([]string, error),
) ([]any, error) {
	keys, err := GraphBFSNodes(
		ctx,
		encode(node),
		depth,
		label,
		func(s string) string { return s },
		expand,
	)
	if err != nil {
		return nil, err
	}

	result := make([]any, len(keys))
	for i, k := range keys {
		result[i] = k
	}

	return result, nil
}
