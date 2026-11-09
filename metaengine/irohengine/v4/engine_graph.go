package irohengine

import (
	"context"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Graph edge replication (CRDT-safe: per-edge LWW register).
//
// An edge is identified by (Collection, From, To). Edge presence resolves by
// last-writer-wins over that identity — identical register semantics to
// MapSet/MapDelete per key — so concurrent or reordered add/remove ops converge
// deterministically: the op with the highest timestamp decides presence, and a
// stale reordered add cannot resurrect an edge removed by a newer remove.
//
// The wire op reuses the existing WriteOp fields: Key carries Edge.From, Value
// carries Edge.To. Graph reads (GraphNeighbors, GraphNeighborsUndirected) stay
// local passthrough — reads never replicate.

// graphLWWKey identifies one edge in the LWW timestamp table. The "graph:"
// prefix and \x1f separator keep edge identities distinct from map-key
// identities sharing the same collection namespace.
func graphLWWKey(from, to any) string {
	return "graph:" + fmt.Sprint(from) + "\x1f" + fmt.Sprint(to)
}

// edgeFromOp rebuilds the edge an op carries. Node endpoints travel as Key and
// Value, so transports that CBOR-round-trip them through `any` need no special
// decoding here.
func edgeFromOp(op WriteOp) metaengine.Edge {
	return metaengine.Edge{From: op.Key, To: op.Value}
}

// writeEdge is the shared body of GraphAddEdge and GraphRemoveEdge: stamp the
// edge's LWW timestamp, apply the local mutation via apply, then publish the
// op for the given kind. Callers assert their own capability first and hand
// the local mutation in as a closure.
func (e *replicatedEngine) writeEdge(
	ctx context.Context,
	collection string,
	edge metaengine.Edge,
	kind OpKind,
	apply func() error,
) error {
	now := e.cfg.clock.Now()
	e.recordLWW(collection, graphLWWKey(edge.From, edge.To), now)
	if err := apply(); err != nil {
		return err
	}
	e.publish(ctx, WriteOp{
		Collection: collection, Kind: kind, Author: e.cfg.author,
		Timestamp: now, Key: edge.From, Value: edge.To,
	})
	return nil
}

// GraphAddEdge applies the edge locally and replicates it to peers. The local
// engine must satisfy the graph dispatch contract (graphCapable); a graphless
// wrapped engine gets ErrGraphBackendNotImplemented rather than silent
// local-only success. graphadapter relies on HasGraphSupport detecting these
// methods structurally.
func (e *replicatedEngine) GraphAddEdge(
	ctx context.Context,
	collection string,
	edge metaengine.Edge,
) error {
	gb, ok := e.local.(graphCapable)
	if !ok {
		return ErrGraphBackendNotImplemented
	}
	return e.writeEdge(ctx, collection, edge, OpGraphAddEdge, func() error {
		return gb.GraphAddEdge(ctx, collection, edge)
	})
}

// GraphRemoveEdge removes the edge locally and replicates the removal. A local
// engine without edge-removal support gets ErrGraphBackendNotImplemented; a
// remote op for such an engine is dropped by applyRemoteGraph (the remove is
// recorded in the LWW table either way, so a stale add still cannot resurrect).
func (e *replicatedEngine) GraphRemoveEdge(
	ctx context.Context,
	collection string,
	edge metaengine.Edge,
) error {
	gx, ok := e.local.(graphExtCapable)
	if !ok {
		return ErrGraphBackendNotImplemented
	}
	return e.writeEdge(ctx, collection, edge, OpGraphRemoveEdge, func() error {
		return gx.GraphRemoveEdge(ctx, collection, edge)
	})
}

// applyRemoteGraphAdd applies an incoming edge-add op under LWW guard.
func (e *replicatedEngine) applyRemoteGraphAdd(op WriteOp) {
	if !e.isLWWNewer(op.Collection, graphLWWKey(op.Key, op.Value), op.Timestamp) {
		return
	}
	if gb, ok := e.local.(graphCapable); ok {
		_ = gb.GraphAddEdge(context.Background(), op.Collection, edgeFromOp(op))
	}
	e.recordLWW(op.Collection, graphLWWKey(op.Key, op.Value), op.Timestamp)
}

// applyRemoteGraphRemove applies an incoming edge-remove op under LWW guard.
// Engines without GraphRemoveEdge drop the apply but still record the
// timestamp, keeping LWW ordering consistent for later ops.
func (e *replicatedEngine) applyRemoteGraphRemove(op WriteOp) {
	if !e.isLWWNewer(op.Collection, graphLWWKey(op.Key, op.Value), op.Timestamp) {
		return
	}
	if gx, ok := e.local.(graphExtCapable); ok {
		_ = gx.GraphRemoveEdge(context.Background(), op.Collection, edgeFromOp(op))
	}
	e.recordLWW(op.Collection, graphLWWKey(op.Key, op.Value), op.Timestamp)
}
