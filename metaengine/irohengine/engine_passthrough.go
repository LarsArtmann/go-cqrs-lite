package irohengine

import (
	"context"
	"errors"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// All methods in this file are LOCAL PASSTHROUGH — they delegate to the wrapped
// engine without replication. These ADTs are either non-CRDT-safe (require
// coordination) or read-only, so they cannot converge via leaderless replication.

// --- ScanBackend (local passthrough) ---

func (e *replicatedEngine) MapScan(
	ctx context.Context, collection string,
	filterFn func(item any) bool,
	sortFunc func(a, b any) int,
	cursor any, limit int,
) (metaengine.ScanResult, error) {
	if sb, ok := e.local.(metaengine.ScanBackend); ok {
		return sb.MapScan(ctx, collection, filterFn, sortFunc, cursor, limit)
	}
	return metaengine.ScanResult{}, errors.New("local engine does not implement ScanBackend")
}

// --- GraphBackend (local passthrough) ---

func (e *replicatedEngine) GraphAddEdge(
	ctx context.Context,
	collection string,
	edge metaengine.Edge,
) error {
	if gb, ok := e.local.(metaengine.GraphBackend); ok {
		return gb.GraphAddEdge(ctx, collection, edge)
	}
	return errors.New("local engine does not implement GraphBackend")
}

func (e *replicatedEngine) GraphNeighbors(
	ctx context.Context,
	collection string,
	node any,
	depth int,
) ([]any, error) {
	if gb, ok := e.local.(metaengine.GraphBackend); ok {
		return gb.GraphNeighbors(ctx, collection, node, depth)
	}
	return nil, errors.New("local engine does not implement GraphBackend")
}

// --- VectorBackend (local passthrough) ---

func (e *replicatedEngine) VectorInsert(
	ctx context.Context,
	collection string,
	emb metaengine.Embedding,
) error {
	if vb, ok := e.local.(metaengine.VectorBackend); ok {
		return vb.VectorInsert(ctx, collection, emb)
	}
	return errors.New("local engine does not implement VectorBackend")
}

func (e *replicatedEngine) VectorSearch(
	ctx context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
) ([]metaengine.VectorResult, error) {
	if vb, ok := e.local.(metaengine.VectorBackend); ok {
		return vb.VectorSearch(ctx, collection, query, k, metric)
	}
	return nil, errors.New("local engine does not implement VectorBackend")
}

// --- SearchBackend (local passthrough) ---

func (e *replicatedEngine) SearchInsert(
	ctx context.Context,
	collection string,
	doc metaengine.IndexedText,
) error {
	if sb, ok := e.local.(metaengine.SearchBackend); ok {
		return sb.SearchInsert(ctx, collection, doc)
	}
	return errors.New("local engine does not implement SearchBackend")
}

func (e *replicatedEngine) SearchQuery(
	ctx context.Context,
	collection, query string,
	limit int,
) ([]metaengine.SearchResult, error) {
	if sb, ok := e.local.(metaengine.SearchBackend); ok {
		return sb.SearchQuery(ctx, collection, query, limit)
	}
	return nil, errors.New("local engine does not implement SearchBackend")
}

// --- SpatialBackend (local passthrough) ---

func (e *replicatedEngine) SpatialInsert(
	ctx context.Context,
	collection string,
	pt metaengine.Point,
) error {
	if sb, ok := e.local.(metaengine.SpatialBackend); ok {
		return sb.SpatialInsert(ctx, collection, pt)
	}
	return errors.New("local engine does not implement SpatialBackend")
}

func (e *replicatedEngine) SpatialRange(
	ctx context.Context,
	collection string,
	x, y, radius float64,
	limit int,
) ([]metaengine.SpatialResult, error) {
	if sb, ok := e.local.(metaengine.SpatialBackend); ok {
		return sb.SpatialRange(ctx, collection, x, y, radius, limit)
	}
	return nil, errors.New("local engine does not implement SpatialBackend")
}
