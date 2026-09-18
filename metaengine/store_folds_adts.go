package metaengine

// Fold application for the specialized ADTs: counters, graph edges, sets,
// multimaps, logs, vectors, search documents, and spatial points.

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func (s *Store) applyFoldCount(
	ctx context.Context,
	q queryMeta,
	fold *countFold,
	rec record.Record,
	payload any,
) error {
	col := q.QueryName()
	delta := fold.invoke(rec, payload)

	if cb, ok := q.QueryEngine().(CounterBackend); ok {
		if err := cb.CounterIncrement(ctx, col, delta); err != nil {
			return fmt.Errorf("counter increment %s: %w", col, err)
		}

		return nil
	}

	return unsupportedEngine(errUnsupportedCounterOps, q.QueryEngine().Profile().Name)
}

func (s *Store) applyFoldEdge(
	ctx context.Context,
	q queryMeta,
	fold *edgeFold,
	rec record.Record,
	payload any,
) error {
	col := q.QueryName()
	edge := fold.invoke(rec, payload)

	if gb, ok := q.QueryEngine().(graphBackend); ok {
		if err := gb.GraphAddEdge(ctx, col, edge); err != nil {
			return fmt.Errorf("graph add edge %s: %w", col, err)
		}

		return nil
	}

	// Degraded fallback: store edge via MultimapBackend (O(N) traversal).
	return graphAddEdgeFallback(ctx, q.QueryEngine(), col, edge)
}

// applyFoldEdgeRemove dispatches tombstone-driven edge removal (ADR-0114
// style): the EdgeRemoval fold retracts a previously added edge. There is no
// degraded fallback — MultimapBackend has no targeted delete — so engines
// without GraphRemoveEdge fail explicitly instead of silently keeping the
// stale edge.
func (s *Store) applyFoldEdgeRemove(
	ctx context.Context,
	q queryMeta,
	fold *edgeRemoveFold,
	rec record.Record,
	payload any,
) error {
	col := q.QueryName()
	removal := fold.invoke(rec, payload)
	eng := q.QueryEngine()

	rm, ok := eng.(graphEdgeRemover)
	if !ok {
		return unsupportedEngine(errEdgeRemoval, eng.Profile().Name)
	}

	if err := rm.GraphRemoveEdge(ctx, col, Edge(removal)); err != nil {
		return fmt.Errorf("graph remove edge %s: %w", col, err)
	}

	return nil
}

func (s *Store) applyFoldSet(
	ctx context.Context,
	q queryMeta,
	fold *setFold,
	rec record.Record,
	payload any,
) error {
	col := q.QueryName()
	key := fold.invoke(rec, payload)

	if sb, ok := q.QueryEngine().(SetBackend); ok {
		if err := sb.SetAdd(ctx, col, key); err != nil {
			return fmt.Errorf("set add %s: %w", col, err)
		}

		return nil
	}

	return unsupportedEngine(errUnsupportedSetOps, q.QueryEngine().Profile().Name)
}

func (s *Store) applyFoldMultiInsert(
	ctx context.Context,
	q queryMeta,
	fold *multiInsertFold,
	rec record.Record,
	payload any,
) error {
	col := q.QueryName()
	entry := fold.invoke(rec, payload)

	if mb, ok := q.QueryEngine().(MultimapBackend); ok {
		if err := mb.MultiAdd(ctx, col, entry.Key, entry.Value); err != nil {
			return fmt.Errorf("multi add %s: %w", col, err)
		}

		return nil
	}

	return unsupportedEngine(errUnsupportedMultimapOps, q.QueryEngine().Profile().Name)
}

func (s *Store) applyFoldAppend(
	ctx context.Context,
	q queryMeta,
	fold *appendFold,
	rec record.Record,
	payload any,
) error {
	col := q.QueryName()
	app := fold.invoke(rec, payload)

	if lb, ok := q.QueryEngine().(LogBackend); ok {
		if err := lb.LogAppend(ctx, col, app.Value); err != nil {
			return fmt.Errorf("log append %s: %w", col, err)
		}

		return nil
	}

	return unsupportedEngine(errUnsupportedLogOps, q.QueryEngine().Profile().Name)
}

func (s *Store) applyFoldVector(
	ctx context.Context,
	q queryMeta,
	fold *vectorFold,
	rec record.Record,
	payload any,
) error {
	col := q.QueryName()
	emb := fold.invoke(rec, payload)

	if vb, ok := q.QueryEngine().(VectorBackend); ok {
		if err := vb.VectorInsert(ctx, col, emb); err != nil {
			return fmt.Errorf("vector insert %s: %w", col, err)
		}

		return nil
	}

	return unsupportedEngine(errUnsupportedVectorOps, q.QueryEngine().Profile().Name)
}

func (s *Store) applyFoldSearch(
	ctx context.Context,
	q queryMeta,
	fold *searchFold,
	rec record.Record,
	payload any,
) error {
	col := q.QueryName()
	doc := fold.invoke(rec, payload)

	if sb, ok := q.QueryEngine().(SearchBackend); ok {
		if err := sb.SearchInsert(ctx, col, doc); err != nil {
			return fmt.Errorf("search insert %s: %w", col, err)
		}

		return nil
	}

	return unsupportedEngine(errUnsupportedSearchOps, q.QueryEngine().Profile().Name)
}

func (s *Store) applyFoldSpatial(
	ctx context.Context,
	q queryMeta,
	fold *spatialFold,
	rec record.Record,
	payload any,
) error {
	col := q.QueryName()
	pt := fold.invoke(rec, payload)

	if sb, ok := q.QueryEngine().(SpatialBackend); ok {
		if err := sb.SpatialInsert(ctx, col, pt); err != nil {
			return fmt.Errorf("spatial insert %s: %w", col, err)
		}

		return nil
	}

	return unsupportedEngine(errUnsupportedSpatialOps, q.QueryEngine().Profile().Name)
}
