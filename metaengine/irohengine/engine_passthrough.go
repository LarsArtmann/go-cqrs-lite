package irohengine

import (
	"context"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// All methods in this file are LOCAL PASSTHROUGH — they delegate to the wrapped
// engine without replication. These ADTs are either non-CRDT-safe (require
// coordination) or read-only, so they cannot converge via leaderless replication.
//
// # Optional-capability forwarding policy (audited 2026-08-16)
//
// The `Replicated` wrapper must decide, per optional capability, whether the
// wrapper promotes the local engine's method set. Interface embedding does NOT
// promote these methods, so each capability is an explicit decision:
//
//   - Closer: FORWARDED (engine.go) — Close shuts down transport first, then
//     the local engine. Both halves always run.
//   - MapUpdater, ScanBackend, VectorBackend, SearchBackend, SpatialBackend,
//     graph dispatch: FORWARDED as local passthrough (this file). Reads are
//     trivially safe; the write-shaped members (VectorInsert, SearchInsert,
//     SpatialInsert, GraphAddEdge) do NOT replicate — the wire protocol has no
//     WriteOp kinds for them; documented on each method.
//   - Transactional (RunInTx): DELIBERATELY NOT FORWARDED. A transaction that
//     writes through the wrapper would replicate per-write (not atomically),
//     and one that writes through the local engine would never replicate.
//     Either way forwarding creates silent divergence, so the wrapper does not
//     implement Transactional and callers see the honest LogBackend path.
//   - StreamLogBackend / SeqSeekableStreamLog / AtomicAppender:
//     DELIBERATELY NOT FORWARDED. StreamAppend is a write the replication
//     protocol does not carry; exposing it would make event streams silently
//     local-only. The system adapters type-assert StreamLogBackend and fall
//     back to the replicated LogBackend (LogAppend/LogTail) path — the
//     degraded route is the correct, converging one.
//   - Prober / TransactMeasurer: DELIBERATELY NOT FORWARDED. A forwarded
//     probe measures local-engine RTT (~0), and live calibration would then
//     override the honest replication-derived NetworkRTT from the wrapper's
//     own latency tracker (latency.go). For replicated engines the transport
//     IS the network hop that matters.

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

// graphExtCapable mirrors the optional graph extension capabilities
// (edge removal + undirected traversal) of the wrapped local engine.
type graphExtCapable interface {
	GraphRemoveEdge(ctx context.Context, collection string, edge metaengine.Edge) error
	GraphNeighborsUndirected(
		ctx context.Context,
		collection string,
		node any,
		depth int,
	) ([]any, error)
}

// GraphRemoveEdge forwards to the local engine WITHOUT replication (same
// wire-protocol limitation as GraphAddEdge — edges do not converge).
func (e *replicatedEngine) GraphRemoveEdge(
	ctx context.Context,
	collection string,
	edge metaengine.Edge,
) error {
	if gx, ok := e.local.(graphExtCapable); ok {
		return gx.GraphRemoveEdge(ctx, collection, edge)
	}
	return ErrGraphBackendNotImplemented
}

// GraphNeighborsUndirected forwards to the local engine. Reads never replicate.
func (e *replicatedEngine) GraphNeighborsUndirected(
	ctx context.Context,
	collection string,
	node any,
	depth int,
) ([]any, error) {
	if gx, ok := e.local.(graphExtCapable); ok {
		return gx.GraphNeighborsUndirected(ctx, collection, node, depth)
	}
	return nil, ErrGraphBackendNotImplemented
}

// --- VectorFilterBackend (local passthrough) ---

func (e *replicatedEngine) VectorSearchFiltered(
	ctx context.Context,
	collection string,
	query []float32,
	k int,
	metric string,
	filters []metaengine.VectorFilter,
) ([]metaengine.VectorResult, error) {
	if vf, ok := e.local.(metaengine.VectorFilterBackend); ok {
		return vf.VectorSearchFiltered(ctx, collection, query, k, metric, filters)
	}
	return nil, ErrVectorBackendNotImplemented
}
