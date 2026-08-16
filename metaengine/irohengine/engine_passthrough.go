package irohengine

import (
	"context"

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
	return metaengine.ScanResult{}, ErrScanBackendNotImplemented
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
	return ErrVectorBackendNotImplemented
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
	return nil, ErrVectorBackendNotImplemented
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
	return ErrSearchBackendNotImplemented
}

func (e *replicatedEngine) SearchQuery(
	ctx context.Context,
	collection, query string,
	limit int,
) ([]metaengine.SearchResult, error) {
	if sb, ok := e.local.(metaengine.SearchBackend); ok {
		return sb.SearchQuery(ctx, collection, query, limit)
	}
	return nil, ErrSearchBackendNotImplemented
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
	return ErrSpatialBackendNotImplemented
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
	return nil, ErrSpatialBackendNotImplemented
}

// --- GraphBackend (local passthrough) ---

// graphCapable mirrors metaengine's unexported graph dispatch contract
// (ADR-0113). The canonical interface is deliberately unexported; this mirror
// exists only to assert and forward the capability of the wrapped engine.
type graphCapable interface {
	GraphAddEdge(ctx context.Context, collection string, edge metaengine.Edge) error
	GraphNeighbors(ctx context.Context, collection string, node any, depth int) ([]any, error)
}

// GraphAddEdge forwards to the local engine WITHOUT replication: the wire
// protocol has no graph WriteOp kind yet, so edges added on one node do not
// converge to peers. Profile() still declares the wrapped engine's graph
// complexity, so the wrapper must forward the dispatch contract structurally —
// graphadapter relies on HasGraphSupport detecting these methods.
func (e *replicatedEngine) GraphAddEdge(
	ctx context.Context,
	collection string,
	edge metaengine.Edge,
) error {
	if gb, ok := e.local.(graphCapable); ok {
		return gb.GraphAddEdge(ctx, collection, edge)
	}
	return ErrGraphBackendNotImplemented
}

// GraphNeighbors forwards to the local engine. Reads never replicate.
func (e *replicatedEngine) GraphNeighbors(
	ctx context.Context,
	collection string,
	node any,
	depth int,
) ([]any, error) {
	if gb, ok := e.local.(graphCapable); ok {
		return gb.GraphNeighbors(ctx, collection, node, depth)
	}
	return nil, ErrGraphBackendNotImplemented
}
