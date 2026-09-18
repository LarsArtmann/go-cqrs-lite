package metaengine

// Query executors for the specialized ADTs: vector k-NN (plain and
// metadata-filtered), undirected graph traversal, full-text search, and
// spatial range queries.

import (
	"context"
	"fmt"
)

func (s *Store) executeVectorSearch(ctx context.Context, q queryMeta, input any) (any, error) {
	queryVec, metric, k, filters := extractVectorQuery(input)

	if len(filters) > 0 {
		return s.executeVectorSearchFiltered(ctx, q, queryVec, k, metric, filters)
	}

	if vb, ok := q.QueryEngine().(VectorBackend); ok {
		results, err := vb.VectorSearch(ctx, q.QueryName(), queryVec, k, metric)
		if err != nil {
			return nil, fmt.Errorf("vector search %s: %w", q.QueryName(), err)
		}

		return results, nil
	}

	return nil, unsupportedEngine(errUnsupportedVectorReads, q.QueryEngine().Profile().Name)
}

// executeVectorSearchFiltered runs metadata-filtered k-NN. Engines with the
// VectorFilterBackend capability filter before ranking, so the k results are
// the k nearest MATCHING neighbors (unlike post-filtering a bare top-k, which
// silently drops results). Plain VectorBackend engines cannot serve filtered
// searches — results carry no metadata — so they fail explicitly instead of
// returning unfiltered rows.
func (s *Store) executeVectorSearchFiltered(
	ctx context.Context,
	q queryMeta,
	queryVec []float32,
	k int,
	metric string,
	filters []VectorFilter,
) (any, error) {
	fb, ok := q.QueryEngine().(VectorFilterBackend)
	if !ok {
		return nil, unsupportedEngine(errUnsupportedVectorFilters, q.QueryEngine().Profile().Name)
	}

	results, err := fb.VectorSearchFiltered(ctx, q.QueryName(), queryVec, k, metric, filters)
	if err != nil {
		return nil, fmt.Errorf("vector search filtered %s: %w", q.QueryName(), err)
	}

	return results, nil
}

// executeGraphNeighborsUndirected serves Undirected:true traversal queries.
// Engines with the GraphNeighborsUndirected capability expand the frontier
// along outgoing AND incoming edges; engines without it fail explicitly
// (silently degrading to directed traversal would return wrong nodes).
func (s *Store) executeGraphNeighborsUndirected(
	ctx context.Context,
	q queryMeta,
	node any,
	depth int,
) (any, error) {
	ub, ok := q.QueryEngine().(undirectedGraphBackend)
	if !ok {
		return nil, unsupportedEngine(errUndirectedGraph, q.QueryEngine().Profile().Name)
	}

	neighbors, err := ub.GraphNeighborsUndirected(ctx, q.QueryName(), node, depth)
	if err != nil {
		return nil, fmt.Errorf("graph neighbors undirected %s: %w", q.QueryName(), err)
	}

	return neighbors, nil
}

func (s *Store) executeFullTextSearch(ctx context.Context, q queryMeta, input any) (any, error) {
	queryText, limit := extractSearchQuery(input)

	if sb, ok := q.QueryEngine().(SearchBackend); ok {
		results, err := sb.SearchQuery(ctx, q.QueryName(), queryText, limit)
		if err != nil {
			return nil, fmt.Errorf("search query %s: %w", q.QueryName(), err)
		}

		return results, nil
	}

	return nil, unsupportedEngine(errUnsupportedSearchReads, q.QueryEngine().Profile().Name)
}

func (s *Store) executeSpatialRange(ctx context.Context, q queryMeta, input any) (any, error) {
	x, y, radius, limit := extractSpatialQuery(input)

	if sb, ok := q.QueryEngine().(SpatialBackend); ok {
		results, err := sb.SpatialRange(ctx, q.QueryName(), x, y, radius, limit)
		if err != nil {
			return nil, fmt.Errorf("spatial range %s: %w", q.QueryName(), err)
		}

		return results, nil
	}

	return nil, unsupportedEngine(errUnsupportedSpatialReads, q.QueryEngine().Profile().Name)
}
