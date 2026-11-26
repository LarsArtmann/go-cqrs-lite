package irohengine

import (
	"context"
	"fmt"
	"sync"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// replicatedEngine wraps a local Engine with CRDT replication.
// Reads always hit the local engine; CRDT-safe writes replicate via Transport.
type replicatedEngine struct {
	local   metaengine.Engine
	cfg     *config
	latency LatencyProvider

	mu         sync.Mutex
	timestamps map[string]time.Time // "col\x00key" → latest LWW timestamp
	pubErr     publishHealth        // outcome of the most recent transport publish
}

func lwwKey(collection string, key any) string {
	return collection + "\x00" + fmt.Sprintf("%v", key)
}

// localBackend asserts the wrapped local engine implements backend T. The
// replicated wrapper forwards every CRDT backend method unconditionally, so
// a local engine missing one is a wiring bug — this names it instead of the
// bare assertion's opaque interface-conversion panic.
func localBackend[T any](e *replicatedEngine) T {
	backend, ok := e.local.(T)
	if !ok {
		panic(fmt.Sprintf("irohengine: local engine %T does not implement %T", e.local, backend))
	}

	return backend
}

func (e *replicatedEngine) isLWWNewer(collection string, key any, ts time.Time) bool {
	k := lwwKey(collection, key) //art-dupl:accept same-file mutex guard idiom

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

// Profile returns an EngineProfile with MEASURED ReplicationLag and NetworkRTT.
// ReplicationLag = P99 of convergence time (last peer to apply).
// NetworkRTT = 2 × P50 of one-way delivery latency.
// Before any traffic, both are zero — no hardcoded guesses.
//
// Cost-model honesty (ADR-0133 sibling note): reads are a LOCAL passthrough
// (documented divergence policy — see modules.md), so the local engine's
// per-pattern ReadCosts remain the correct read prices and are inherited
// unchanged. Replication cost surfaces as ReplicationLag/NetworkRTT above,
// NOT as inflated read/write scalars: writes are async leaderless CRDT
// applies, so a local write price plus observed convergence is the truthful
// model, and a synchronous-write surcharge would misprice the planner.
func (e *replicatedEngine) Profile() metaengine.EngineProfile {
	p := e.local.Profile()
	p.Name = "iroh(" + p.Name + ")"
	p.Replication = metaengine.ReplicationLeaderless

	if e.latency != nil {
		snap := e.latency.LatencySnapshot()
		p.ReplicationLag = snap.ConvergenceP99
		p.NetworkRTT = snap.DeliveryP50 * 2
	}
	// ADR-0142 write-side refusal (never silence): the wrapper does NOT
	// forward DueClaimer/DedupStore — a lease or dedup window taken on one
	// replica is invisible to its peers until convergence, so leaderless
	// replication cannot fence concurrent claimers (double-fire) or hold a
	// check-and-set window. The local engine still implements the
	// capabilities (un-wrapped); the wrapper refuses them explicitly so
	// operator engine picks never route claims/dedup through replication.
	for _, adt := range []metaengine.ADT{metaengine.ADTDueClaim, metaengine.ADTDedup} {
		if _, supported := p.Supports[adt]; supported {
			delete(p.Supports, adt)
			delete(p.DegradedADTs, adt)
		}

		if p.RefusedADTs == nil {
			p.RefusedADTs = make(map[metaengine.ADT]string, 2)
		}

		p.RefusedADTs[adt] = "leaderless replication cannot fence concurrent claimers — claims/dedup must not ride the replicated wrapper"
	}

	return p
}

// --- MapBackend (CRDT-safe: LWW-Map) ---

func (e *replicatedEngine) MapSet(
	ctx context.Context,
	collection string,
	key any,
	value any,
) error {
	now := e.cfg.clock.Now()
	e.recordLWW(collection, key, now)

	if err := localBackend[metaengine.MapBackend](e).MapSet(ctx, collection, key, value); err != nil {
		return err
	}

	e.publish(ctx, WriteOp{
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
	return localBackend[metaengine.MapBackend](e).MapGet(ctx, collection, key)
}

func (e *replicatedEngine) MapDelete(ctx context.Context, collection string, key any) error {
	now := e.cfg.clock.Now()
	e.recordLWW(collection, key, now)

	if err := localBackend[metaengine.MapBackend](e).MapDelete(ctx, collection, key); err != nil {
		return err
	}

	e.publish(ctx, WriteOp{
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

	return ErrMapUpdaterNotImplemented
}

// --- SetBackend (CRDT-safe: OR-Set) ---

func (e *replicatedEngine) SetAdd(ctx context.Context, collection string, key any) error {
	if err := localBackend[metaengine.SetBackend](e).SetAdd(ctx, collection, key); err != nil {
		return err
	}

	e.publish(ctx, WriteOp{
		Collection: collection, Kind: OpSetAdd, Author: e.cfg.author,
		Timestamp: e.cfg.clock.Now(), Key: key,
	})

	return nil
}

func (e *replicatedEngine) SetContains(
	ctx context.Context,
	collection string,
	key any,
) (bool, error) {
	return localBackend[metaengine.SetBackend](e).SetContains(ctx, collection, key)
}

// --- CounterBackend (CRDT-safe: PN-Counter) ---

func (e *replicatedEngine) CounterIncrement(
	ctx context.Context,
	collection string,
	deltas metaengine.Delta,
) error {
	if err := localBackend[metaengine.CounterBackend](e).CounterIncrement(
		ctx,
		collection,
		deltas,
	); err != nil {
		return err
	}

	e.publish(ctx, WriteOp{
		Collection: collection, Kind: OpCounterInc, Author: e.cfg.author,
		Timestamp: e.cfg.clock.Now(), Delta: deltas,
	})

	return nil
}

func (e *replicatedEngine) CounterGet(
	ctx context.Context,
	collection string,
) (map[string]int64, error) {
	return localBackend[metaengine.CounterBackend](e).CounterGet(ctx, collection)
}

// --- MultimapBackend (CRDT-safe: OR-Set per key) ---

func (e *replicatedEngine) MultiAdd(
	ctx context.Context,
	collection string,
	key any,
	value any,
) error {
	if err := localBackend[metaengine.MultimapBackend](e).MultiAdd(
		ctx,
		collection,
		key,
		value,
	); err != nil {
		return err
	}

	e.publish(ctx, WriteOp{
		Collection: collection, Kind: OpMultiAdd, Author: e.cfg.author,
		Timestamp: e.cfg.clock.Now(), Key: key, Value: value,
	})

	return nil
}

func (e *replicatedEngine) MultiGet(
	ctx context.Context,
	collection string,
	key any,
) ([]any, error) {
	return localBackend[metaengine.MultimapBackend](e).MultiGet(ctx, collection, key)
}

// --- LogBackend (CRDT-safe: per-author append-only) ---

func (e *replicatedEngine) LogAppend(ctx context.Context, collection string, value any) error {
	if err := localBackend[metaengine.LogBackend](e).LogAppend(ctx, collection, value); err != nil {
		return err
	}

	e.publish(ctx, WriteOp{
		Collection: collection, Kind: OpLogAppend, Author: e.cfg.author,
		Timestamp: e.cfg.clock.Now(), Value: value,
	})

	return nil
}

func (e *replicatedEngine) LogTail(
	ctx context.Context,
	collection string,
	limit int,
) ([]any, error) {
	return localBackend[metaengine.LogBackend](e).LogTail(ctx, collection, limit)
}

// Compile-time assertions that replicatedEngine implements all backend interfaces.
var (
	_ metaengine.Engine          = (*replicatedEngine)(nil)
	_ metaengine.MapBackend      = (*replicatedEngine)(nil)
	_ metaengine.MapUpdater      = (*replicatedEngine)(nil)
	_ metaengine.ScanBackend     = (*replicatedEngine)(nil)
	_ metaengine.SetBackend      = (*replicatedEngine)(nil)
	_ metaengine.CounterBackend  = (*replicatedEngine)(nil)
	_ metaengine.MultimapBackend = (*replicatedEngine)(nil)
	_ metaengine.LogBackend      = (*replicatedEngine)(nil)
	_ metaengine.VectorBackend   = (*replicatedEngine)(nil)
	_ metaengine.SearchBackend   = (*replicatedEngine)(nil)
	_ metaengine.SpatialBackend  = (*replicatedEngine)(nil)
)
