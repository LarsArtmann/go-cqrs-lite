package irohengine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// replicatedEngine wraps a local Engine with CRDT replication.
// Reads always hit the local engine; CRDT-safe writes replicate via Transport.
type replicatedEngine struct {
	local metaengine.Engine
	cfg   *config

	mu         sync.Mutex
	timestamps map[string]time.Time // "col\x00key" → latest LWW timestamp
	applying   bool                 // re-entrancy guard: true when applying remote op
}

func lwwKey(collection string, key any) string {
	return collection + "\x00" + fmt.Sprintf("%v", key)
}

func (e *replicatedEngine) isLWWNewer(collection string, key any, ts time.Time) bool {
	k := lwwKey(collection, key)
	e.mu.Lock()
	defer e.mu.Unlock()
	existing := e.timestamps[k]
	return ts.After(existing) || ts.Equal(existing)
}

func (e *replicatedEngine) recordLWW(collection string, key any, ts time.Time) {
	k := lwwKey(collection, key)
	e.mu.Lock()
	defer e.mu.Unlock()
	if ts.After(e.timestamps[k]) {
		e.timestamps[k] = ts
	}
}

// --- Engine + Closer ---

func (e *replicatedEngine) Profile() metaengine.EngineProfile {
	p := e.local.Profile()
	p.Name = "iroh(" + p.Name + ")"
	p.Replication = metaengine.ReplicationLeaderless
	p.ReplicationLag = e.cfg.replicationLag
	p.NetworkRTT = e.cfg.networkRTT
	return p
}

func (e *replicatedEngine) Close() error {
	if e.cfg.transport != nil {
		_ = e.cfg.transport.Close()
	}
	return e.local.Close()
}

// publish sends a WriteOp to remote nodes (skip when applying remote or no transport).
func (e *replicatedEngine) publish(op WriteOp) {
	if e.cfg.transport == nil {
		return
	}
	e.mu.Lock()
	skip := e.applying
	e.mu.Unlock()
	if skip {
		return
	}
	_ = e.cfg.transport.Publish(context.Background(), op)
}

// applyRemote dispatches an incoming WriteOp to the local engine.
func (e *replicatedEngine) applyRemote(op WriteOp) {
	e.mu.Lock()
	e.applying = true
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		e.applying = false
		e.mu.Unlock()
	}()

	ctx := context.Background()

	switch op.Kind {
	case OpMapSet:
		if !e.isLWWNewer(op.Collection, op.Key, op.Timestamp) {
			return
		}
		if mb, ok := e.local.(metaengine.MapBackend); ok {
			_ = mb.MapSet(ctx, op.Collection, op.Key, op.Value)
		}
		e.recordLWW(op.Collection, op.Key, op.Timestamp)

	case OpMapDelete:
		if !e.isLWWNewer(op.Collection, op.Key, op.Timestamp) {
			return
		}
		if mb, ok := e.local.(metaengine.MapBackend); ok {
			_ = mb.MapDelete(ctx, op.Collection, op.Key)
		}
		e.recordLWW(op.Collection, op.Key, op.Timestamp)

	case OpSetAdd:
		if sb, ok := e.local.(metaengine.SetBackend); ok {
			_ = sb.SetAdd(ctx, op.Collection, op.Key)
		}

	case OpCounterInc:
		if cb, ok := e.local.(metaengine.CounterBackend); ok {
			_ = cb.CounterIncrement(ctx, op.Collection, op.Delta)
		}

	case OpMultiAdd:
		if mb, ok := e.local.(metaengine.MultimapBackend); ok {
			_ = mb.MultiAdd(ctx, op.Collection, op.Key, op.Value)
		}

	case OpLogAppend:
		if lb, ok := e.local.(metaengine.LogBackend); ok {
			_ = lb.LogAppend(ctx, op.Collection, op.Value)
		}
	}
}

// --- MapBackend (CRDT-safe: LWW-Map) ---

func (e *replicatedEngine) MapSet(
	ctx context.Context,
	collection string,
	key any,
	value any,
) error {
	now := time.Now()
	e.recordLWW(collection, key, now)
	if err := e.local.(metaengine.MapBackend).MapSet(ctx, collection, key, value); err != nil {
		return err
	}
	e.publish(WriteOp{
		Collection: collection, Kind: OpMapSet, Author: e.cfg.author,
		Timestamp: now, Key: key, Value: value,
	})
	return nil
}

func (e *replicatedEngine) MapGet(
	ctx context.Context,
	collection string,
	key any,
) (any, bool, error) {
	return e.local.(metaengine.MapBackend).MapGet(ctx, collection, key)
}

func (e *replicatedEngine) MapDelete(ctx context.Context, collection string, key any) error {
	now := time.Now()
	e.recordLWW(collection, key, now)
	if err := e.local.(metaengine.MapBackend).MapDelete(ctx, collection, key); err != nil {
		return err
	}
	e.publish(WriteOp{
		Collection: collection, Kind: OpMapDelete, Author: e.cfg.author,
		Timestamp: now, Key: key,
	})
	return nil
}

// --- MapUpdater (NOT CRDT-safe: local only) ---

func (e *replicatedEngine) MapUpdate(
	ctx context.Context,
	collection string,
	key any,
	update func(prev any) any,
) error {
	if mu, ok := e.local.(metaengine.MapUpdater); ok {
		return mu.MapUpdate(ctx, collection, key, update)
	}
	return errors.New("local engine does not implement MapUpdater")
}

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

// --- SetBackend (CRDT-safe: OR-Set) ---

func (e *replicatedEngine) SetAdd(ctx context.Context, collection string, key any) error {
	if err := e.local.(metaengine.SetBackend).SetAdd(ctx, collection, key); err != nil {
		return err
	}
	e.publish(WriteOp{
		Collection: collection, Kind: OpSetAdd, Author: e.cfg.author,
		Timestamp: time.Now(), Key: key,
	})
	return nil
}

func (e *replicatedEngine) SetContains(
	ctx context.Context,
	collection string,
	key any,
) (bool, error) {
	return e.local.(metaengine.SetBackend).SetContains(ctx, collection, key)
}

// --- CounterBackend (CRDT-safe: PN-Counter) ---

func (e *replicatedEngine) CounterIncrement(
	ctx context.Context,
	collection string,
	deltas metaengine.Delta,
) error {
	if err := e.local.(metaengine.CounterBackend).CounterIncrement(
		ctx,
		collection,
		deltas,
	); err != nil {
		return err
	}
	e.publish(WriteOp{
		Collection: collection, Kind: OpCounterInc, Author: e.cfg.author,
		Timestamp: time.Now(), Delta: deltas,
	})
	return nil
}

func (e *replicatedEngine) CounterGet(
	ctx context.Context,
	collection string,
) (map[string]int64, error) {
	return e.local.(metaengine.CounterBackend).CounterGet(ctx, collection)
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

// --- MultimapBackend (CRDT-safe: OR-Set per key) ---

func (e *replicatedEngine) MultiAdd(
	ctx context.Context,
	collection string,
	key any,
	value any,
) error {
	if err := e.local.(metaengine.MultimapBackend).MultiAdd(
		ctx,
		collection,
		key,
		value,
	); err != nil {
		return err
	}
	e.publish(WriteOp{
		Collection: collection, Kind: OpMultiAdd, Author: e.cfg.author,
		Timestamp: time.Now(), Key: key, Value: value,
	})
	return nil
}

func (e *replicatedEngine) MultiGet(
	ctx context.Context,
	collection string,
	key any,
) ([]any, error) {
	return e.local.(metaengine.MultimapBackend).MultiGet(ctx, collection, key)
}

// --- LogBackend (CRDT-safe: per-author append-only) ---

func (e *replicatedEngine) LogAppend(ctx context.Context, collection string, value any) error {
	if err := e.local.(metaengine.LogBackend).LogAppend(ctx, collection, value); err != nil {
		return err
	}
	e.publish(WriteOp{
		Collection: collection, Kind: OpLogAppend, Author: e.cfg.author,
		Timestamp: time.Now(), Value: value,
	})
	return nil
}

func (e *replicatedEngine) LogTail(
	ctx context.Context,
	collection string,
	limit int,
) ([]any, error) {
	return e.local.(metaengine.LogBackend).LogTail(ctx, collection, limit)
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

// Compile-time assertions that replicatedEngine implements all backend interfaces.
var (
	_ metaengine.Engine          = (*replicatedEngine)(nil)
	_ metaengine.MapBackend      = (*replicatedEngine)(nil)
	_ metaengine.MapUpdater      = (*replicatedEngine)(nil)
	_ metaengine.ScanBackend     = (*replicatedEngine)(nil)
	_ metaengine.SetBackend      = (*replicatedEngine)(nil)
	_ metaengine.CounterBackend  = (*replicatedEngine)(nil)
	_ metaengine.GraphBackend    = (*replicatedEngine)(nil)
	_ metaengine.MultimapBackend = (*replicatedEngine)(nil)
	_ metaengine.LogBackend      = (*replicatedEngine)(nil)
	_ metaengine.VectorBackend   = (*replicatedEngine)(nil)
	_ metaengine.SearchBackend   = (*replicatedEngine)(nil)
	_ metaengine.SpatialBackend  = (*replicatedEngine)(nil)
)
