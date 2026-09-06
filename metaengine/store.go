package metaengine

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

type Store struct {
	mu                sync.RWMutex
	foldLocks         *foldLocks // per-query fold locks (shared fold state; see fold_locks.go)
	engines           []Engine
	engineRoles       map[string]ProjectionRole             // engine name → role; missing = Active (guarded by mu)
	replicas          map[string]*replicator                // shadow-engine replicators (guarded by mu)
	taskSnap          atomic.Pointer[map[string][]foldTask] // immutable event→tasks index (lock-free reads)
	queries           map[string]queryMeta
	byInputType       map[string]string
	plan              *PlanResult
	poison            *poisonTracker
	idempotency       *idempotencyTracker
	meter             *workloadMeter
	subs              *subscriberHub
	hooks             *Hooks // observability hooks (nil = no-op)
	eventLog          *EventLog
	queryDecls        []any          // original query declarations (for Verify)
	coalescer         *ReadCoalescer // optional read coalescer (nil = disabled)
	routingHysteresis float64        // min fractional improvement before suggesting re-route
	routingMinDelta   float64        // min absolute improvement (ms) before suggesting re-route
	lastReplanAt      time.Time
	replanCount       int
	planHistory       []PlanAuditEntry // bounded audit trail (max maxPlanHistory)
	routingMu         sync.Mutex       // protects routingSig + routingDiags
	routingSig        string
	routingDiags      []Diagnostic
	priorityConfig    *PriorityConfig // operator-driven layout priority (ADR-0124)
	sharedCollections map[string]bool // child types shared across collections (ADR-0124 boundaries)

	// Record-context hazard tracking: applies that arrived as a synthesized
	// Type-only Record (Store.Apply) while OnRecord folds were registered for
	// the event type. See record_context.go.
	recordAwareEvents       atomic.Pointer[map[string]bool]
	syntheticRecordApplies  atomic.Uint64
	syntheticRecordAdvisory sync.Once
}

func (s *Store) Plan() *PlanResult { return s.plan }

// Replan recomputes the plan using current engine profiles. This is the primary
// mechanism for picking up live latency measurements: ProbeEngine continuously
// updates trackers via background probing, and Replan re-reads Profile() which
// reflects those updates through ApplyCalibration. Call Replan periodically
// (e.g. every 30s) or after detecting a significant latency shift.
//
// Replan is safe for concurrent use: it holds the Store's write lock only
// during engine re-assignment (mutating QueryDecl) and the atomic plan swap.
// The rule pipeline runs without the lock — same as Plan() — because rules
// read from the Store and would self-deadlock if the write lock were held.
//
// The plan version is incremented on each successful Replan. Consumers can
// compare versions to detect that a re-plan occurred without inspecting the
// full PlanResult.
func (s *Store) Replan(ctx context.Context) error {
	return s.replanWithTrigger(ctx, triggerManual)
}

// replanWithTrigger is the shared body of every re-plan path. The trigger
// string is recorded in the audit trail so operators can distinguish a manual
// Replan from one caused by SetPriority, AddEngine/RemoveEngine, or the
// auto-reroute loop.
func (s *Store) replanWithTrigger(ctx context.Context, trigger string) error {
	return s.replanWithTransition(ctx, trigger, nil)
}

// replanWithTransition re-plans with an optional transition hook executed at
// the start of Phase 1 under the SAME store write lock that re-assigns query
// engines. Role transitions (PromoteEngine, DemoteEngine) use the hook so the
// role flip, the replicator swap, and the assignment mutation are atomic:
// concurrent events (dispatch + replication under one read lock) either see
// the world entirely before the transition or entirely after it, so an engine
// receives each event exactly once, never twice, never zero times
// (ADR-0124 §7, METAENGINE-LAYOUT-ROLES.md §4).
//
// A hook error aborts the re-plan before any mutation happens.
func (s *Store) replanWithTransition(
	ctx context.Context,
	trigger string,
	underLock func() error,
) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("metaengine.Store.Replan: %w", err)
	}

	cfg := planConfig{
		writeAmplificationBudget: DefaultWriteAmplificationBudget,
		priority:                 s.priorityConfig,
		sharedCollections:        s.sharedCollections,
		routingHysteresis:        s.routingHysteresis,
		routingMinDeltaMs:        s.routingMinDelta,
	}

	// Phase 1: re-assign engines under the write lock (mutates QueryDecl).
	// Only routable engines (Active/DualUse) are candidates — shadow engines
	// (Backup/Migration) never serve reads (METAENGINE-LAYOUT-ROLES.md I1).
	plan := &PlanResult{}

	s.mu.Lock()

	if underLock != nil {
		if err := underLock(); err != nil {
			s.mu.Unlock()
			return fmt.Errorf("metaengine.Store.Replan: %w", err)
		}
	}

	routable := s.routableLocked()

	// Incumbent awareness (C5.3): without it, Replan re-assigns to the argmin
	// even for sub-hysteresis improvements, so two near-parity engines
	// oscillate A→B→A across successive auto-replan ticks.
	if s.plan != nil {
		incumbents := make(map[string]string, len(s.plan.Queries))
		for _, qa := range s.plan.Queries {
			incumbents[qa.QueryName] = qa.EngineName
		}
		cfg.incumbents = incumbents
	}

	for _, name := range slices.Sorted(maps.Keys(s.queries)) {
		q := s.queries[name]

		assignment, err := planQuery(q, routable, cfg)
		if err != nil {
			s.mu.Unlock()
			return fmt.Errorf("metaengine.Store.Replan: %w", err)
		}

		plan.Queries = append(plan.Queries, assignment)
	}
	s.mu.Unlock()

	// Phase 2: run rules without the lock (rules read from Store and would
	// self-deadlock if the write lock were held — same pattern as Plan()).
	pipeline := NewRulePipeline(defaultRules(cfg)...)
	if err := pipeline.Apply(plan, PlanContext{Store: s, Config: cfg}); err != nil {
		return fmt.Errorf("metaengine.Store.Replan: %w", err)
	}

	// Phase 3: atomically swap the plan under the write lock.
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.plan != nil {
		plan.Version = s.plan.Version + 1
	} else {
		plan.Version = 1
	}

	plan.ComputedAt = time.Now()
	s.plan = plan
	s.lastReplanAt = plan.ComputedAt
	s.replanCount++
	s.appendPlanAudit(plan.Version, plan.ComputedAt, trigger, s.priorityConfig)

	slog.Info("metaengine: replan completed",
		"version", plan.Version, "queries", len(plan.Queries), "trigger", trigger)

	return nil
}

// CollectionInfo describes a planned query collection.
type CollectionInfo struct {
	Name        string
	ADT         ADT
	ReadPattern ReadPattern
	EngineName  string
	Complexity  Complexity

	// Persistence declares whether the collection's data survives process
	// exit (DDIA Ch1: survivability). PersistenceVolatile means data is lost
	// on restart and must be rebuilt from the event log.
	Persistence Persistence

	// Replication declares how the collection's engine propagates data
	// across process boundaries (DDIA Ch5). ReplicationNone means single-node.
	Replication Replication

	// ReplicationLagMs is the expected staleness for replicated engines,
	// in milliseconds. Zero for single-node engines. Diagnostic-only.
	ReplicationLagMs int64

	// NetworkRTTMs is the additive fixed network latency, in milliseconds.
	NetworkRTTMs int64
}

// Collections returns metadata for every registered query collection.
// The result is sorted by collection name.
func (s *Store) Collections() []CollectionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]CollectionInfo, 0, len(s.queries))

	for _, name := range slices.Sorted(maps.Keys(s.queries)) {
		q := s.queries[name]
		profile := q.QueryEngine().Profile()
		result = append(result, CollectionInfo{
			Name:             q.QueryName(),
			ADT:              q.QueryADT(),
			ReadPattern:      q.QueryReadPattern(),
			EngineName:       profile.Name,
			Complexity:       q.QueryComplexity(),
			Persistence:      profile.Persistence,
			Replication:      profile.Replication,
			ReplicationLagMs: profile.EffectiveReplicationLag().Milliseconds(),
			NetworkRTTMs:     profile.EffectiveNetworkRTT().Milliseconds(),
		})
	}

	return result
}

// lookupQuery returns the queryMeta for a collection name under a read lock.
// queryMeta values are immutable after Plan(), so the returned value remains
// valid after the lock is released.
func (s *Store) lookupQuery(name string) (queryMeta, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	q, ok := s.queries[name]
	return q, ok
}

// enginesSnapshot returns the registered engines slice header under the read
// lock. The returned slice must not be mutated. Shared by diagnostics paths
// (GetEngineStats, Doctor) that iterate engines outside the lock.
func (s *Store) enginesSnapshot() []Engine {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.engines
}

// collectionEngine returns the engine assigned to a query/collection by name.
// Used by TypedReader to access the engine for typed reads without going through
// the reflective Execute path.
func (s *Store) collectionEngine(collection string) (Engine, bool) {
	q, ok := s.lookupQuery(collection)
	if !ok {
		return nil, false
	}

	return q.QueryEngine(), true
}

// ReplicationMode returns the replication topology for a query's collection.
// Returns ReplicationNone if the collection doesn't exist or the engine is
// single-node. Use this to decide whether read-after-write consistency
// guarantees apply (they don't for replicated engines with non-zero lag).
func (s *Store) ReplicationMode(queryName string) Replication {
	q, ok := s.lookupQuery(queryName)
	if !ok {
		return ReplicationNone
	}

	return q.QueryEngine().Profile().Replication
}

// Persistence returns the durability classification for a query's collection.
// Returns PersistenceVolatile if the collection doesn't exist or the engine
// is RAM-backed. Use this to decide whether a restart will lose the projection
// (requiring a replay from the event log).
func (s *Store) Persistence(queryName string) Persistence {
	q, ok := s.lookupQuery(queryName)
	if !ok {
		return PersistenceVolatile
	}

	return q.QueryEngine().Profile().Persistence
}

// IsPoisoned returns the poison error if the collection was poisoned by a fold
// panic, or nil if healthy. Once poisoned, a collection refuses reads until
// the store is recreated (or the poison is cleared via Reset).
func (s *Store) IsPoisoned(collection string) error {
	return s.poison.Check(collection)
}

// EventTypes returns every event type that at least one registered query
// listens to. The result is sorted for deterministic ordering.
// Used by integration adapters (e.g. projectionadapter) that need to
// declare their projection's event interests without depending on
// event-sourcing packages directly.
func (s *Store) EventTypes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[string]struct{})

	for _, q := range s.queries {
		for et := range q.QueryFoldByEvent() {
			seen[et] = struct{}{}
		}
	}

	result := make([]string, 0, len(seen))
	for et := range seen {
		result = append(result, et)
	}

	slices.Sort(result)

	return result
}

func (s *Store) Close() error {
	s.mu.Lock()
	reps := make([]*replicator, 0, len(s.replicas))

	for _, rep := range s.replicas {
		reps = append(reps, rep)
	}

	s.replicas = nil
	s.mu.Unlock()

	for _, rep := range reps {
		rep.halt()
	}

	var firstErr error
	for _, eng := range s.engines {
		if err := eng.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// registerWatcher adds a watcher entry for a collection. Called by
// Watcher.Watch during setup. Thread-safe.
func (s *Store) registerWatcher(collection string, entry *watcherEntry) {
	s.subs.registerWatcher(collection, entry)
}

// registerReplay attaches a replay recorder to a collection so that
// notifyWatchers records each value change with a monotonic sequence number.
// Called by Watcher.WithReplay. Thread-safe.
func (s *Store) registerReplay(collection string, recorder replayRecorder) {
	s.subs.registerReplay(collection, recorder)
}

// unregisterReplay removes the replay recorder for a collection.
// Called by Watcher.Close. Thread-safe.
func (s *Store) unregisterReplay(collection string) {
	s.subs.unregisterReplay(collection)
}

// keysMatch compares two keys for watcher filtering. Uses reflect.DeepEqual
// for correctness across types (strings, ints, branded IDs).
func keysMatch(a, b any) bool {
	return reflect.DeepEqual(a, b)
}

// notifyWatchers delegates to the subscriber hub.
func (s *Store) notifyWatchers(collection string, key any, value any) {
	s.subs.notify(collection, key, value)
}

// notifyLive notifies watchers unless the write comes from the replication
// shim (shadow engine): primaries already notified, and replaying a shadow
// write would double-append watcher replay sequences (METAENGINE-LAYOUT-ROLES §3.2).
func (s *Store) notifyLive(q queryMeta, collection string, key any, value any) {
	if q.isShadow() {
		return
	}

	s.notifyWatchers(collection, key, value)
}

// Apply processes an event through ALL queries that listen to it.
// Each query has its own independent projection — the same event updates
// each matching query's collection separately.
//
// The Record handed to folds is SYNTHESIZED: only Type is set — StreamID,
// Version, and metadata are zero. OnRecord handlers that read per-instance
// context must be fed via ApplyRecord instead; Store counts such applies
// and Doctor's "--- Record context ---" section reports them.
func (s *Store) Apply(ctx context.Context, eventType string, payload any) error {
	return s.applyWithRecord(ctx, eventType, record.Record{Type: eventType}, payload)
}

// EventInput pairs an event type with its payload for batch application.
// Record optionally carries the full record context: when set, replay paths
// (Backfill, Verify, DemoteEngine catch-up) rebuild Record-aware projections
// with the original StreamID/Version/metadata instead of a synthesized
// minimal record.
type EventInput struct {
	Type    string
	Payload any
	Record  record.Record
}

// ApplyBatch processes multiple events through all queries in one call.
// Events are applied sequentially; on the first error, remaining events are
// skipped and the error is returned. This is the primary API for replay
// scenarios where many events need to be processed.
//
// Events without a Record get the same synthesized Type-only Record as
// Store.Apply — set EventInput.Record for OnRecord folds that read context.
func (s *Store) ApplyBatch(ctx context.Context, events []EventInput) error {
	for _, evt := range events {
		if err := s.Apply(ctx, evt.Type, evt.Payload); err != nil {
			return fmt.Errorf("batch apply event %q: %w", evt.Type, err)
		}
	}

	return nil
}

// ApplyRecord processes a decoded payload with full Record context (ADR-0112).
// Record-aware folds (created via OnRecord) receive the full Record — StreamID,
// Version, MetaData — alongside the payload. Non-Record-aware folds (created via
// On) receive only the payload, as usual. This is the ONLY entry point that
// provides per-instance context: Apply synthesizes a Type-only Record.
//
// The decodedPayload is the already-decoded Go struct (e.g. UserCreated{...}),
// not raw bytes. The Record carries metadata context; the payload carries the
// domain data.
func (s *Store) ApplyRecord(
	ctx context.Context,
	rec record.Record,
	decodedPayload any,
) error {
	return s.applyWithRecord(ctx, rec.Type, rec, decodedPayload)
}

// applyWithRecord dispatches a payload through all matching folds, setting the
// Record context on RecordAwareFold implementations before invoke.
//
// Fold operations are grouped by engine and applied atomically: when an engine
// implements Transactional, all its fold operations for this event execute in a
// single RunInTx. This ensures that if one fold fails, the engine's transaction
// rolls back — preserving the invariant that an event is an atomic batch boundary.
// Cross-engine atomicity is NOT guaranteed (two-phase commit is not supported).
func (s *Store) applyWithRecord(
	ctx context.Context,
	eventType string,
	rec record.Record,
	payload any,
) (err error) {
	start := time.Now()

	defer func() {
		if s.hooks != nil && s.hooks.OnApply != nil {
			s.hooks.OnApply(eventType, time.Since(start), err)
		}
	}()

	s.meter.IncWrite()

	// Record, dispatch, and replicate under ONE read-lock section so a
	// concurrent role transition (PromoteEngine/DemoteEngine, write-locked)
	// can never land between them: every event reaches each engine either as
	// a primary fold or via replication, never both and never neither, and
	// the transition's EventLog snapshot splits history at exactly the same
	// point the routing flips.
	s.mu.RLock()

	if s.eventLog != nil {
		s.eventLog.RecordEvent(eventType, rec, payload)
	}

	if isSyntheticRecord(rec) {
		s.noteSyntheticRecordApply(eventType)
	}

	dispatchErr := s.dispatchFoldsLocked(ctx, eventType, rec, payload, nil)
	if dispatchErr == nil {
		// Replication follows APPLIED state (not the event log): only
		// successful primary dispatches are mirrored.
		s.replicateLocked(eventType, rec, payload)
	}
	s.mu.RUnlock()

	if dispatchErr != nil {
		return dispatchErr
	}

	return nil
}

// replicateLocked fans a successfully applied event out to all shadow
// engines. Non-blocking: failure isolation is a design invariant (I3), a slow
// or broken mirror must never stall the primary write path. The caller must
// hold s.mu (at least RLock).
func (s *Store) replicateLocked(eventType string, rec record.Record, payload any) {
	for _, rep := range s.replicas {
		rep.tryEnqueue(repJob{eventType: eventType, rec: rec, payload: payload})
	}
}

// ApplyIdempotent processes an event with deduplication by event ID. If the
// same eventID has been applied before, the call is a no-op. This is essential
// for at-least-once delivery scenarios where events may be replayed.
//
// The dedup is in-memory (process-local); for durable dedup across restarts,
// consumers should wrap the Store with an external idempotency store.
func (s *Store) ApplyIdempotent(ctx context.Context, eventID, eventType string, payload any) error {
	if eventID == "" {
		return s.Apply(ctx, eventType, payload)
	}

	if s.idempotency.CheckAndRecord(eventID) {
		return nil // already applied
	}

	return s.Apply(ctx, eventType, payload)
}

// InTransaction executes fn within a single database transaction across all
// engines that support transactions (via the Transactional interface). If any
// engine's RunInTx fails, the error is returned immediately. Engines that do
// not implement Transactional (e.g. memory engine) execute fn normally
// without transaction semantics.
//
// Usage:
//
//	err := store.InTransaction(ctx, func(ctx context.Context) error {
//	    if err := store.Apply(ctx, "user.created", payload); err != nil {
//	        return err
//	    }
//	    return store.Apply(ctx, "profile.updated", profile)
//	})
//
// Note: with multiple transactional engines, each gets its own transaction.
// Cross-engine atomicity is NOT guaranteed (two-phase commit is not supported).
func (s *Store) InTransaction(ctx context.Context, fn func(context.Context) error) error {
	// Find the first transactional engine and delegate to it.
	for _, eng := range s.engines {
		if tx, ok := eng.(Transactional); ok {
			return tx.RunInTx(ctx, fn) //nolint:wrapcheck
		}
	}

	// No transactional engine — just run fn.
	return fn(ctx)
}

func (s *Store) applyFold(
	ctx context.Context,
	q queryMeta,
	fold Fold,
	rec record.Record,
	payload any,
) (err error) {
	start := time.Now()

	// Wrap errors with structured context for debugging. Registered first so
	// it runs last (LIFO) — hooks and panic recovery see the raw error.
	defer func() {
		if err != nil {
			err = &ApplyError{
				Query:     q.QueryName(),
				EventType: fold.EventType(),
				FoldKind:  fold.Kind(),
				Cause:     err,
			}
		}
	}()

	defer func() {
		if s.hooks != nil && s.hooks.OnFold != nil {
			s.hooks.OnFold(q.QueryName(), fold.EventType(), fold.Kind(), time.Since(start), err)
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			poisonErr := fmt.Errorf("%w: collection %q, panic: %v", ErrPoisoned, q.QueryName(), r)
			s.poison.Poison(q.QueryName(), poisonErr)
			err = poisonErr
		}
	}()

	switch f := fold.(type) {
	case *insertFold:
		return s.applyFoldInsert(ctx, q, f, rec, payload)
	case *updateFold:
		return s.applyFoldUpdate(ctx, q, f, rec, payload)
	case *removeFold:
		return s.applyFoldRemove(ctx, q, f, rec, payload)
	case *countFold:
		return s.applyFoldCount(ctx, q, f, rec, payload)
	case *edgeFold:
		return s.applyFoldEdge(ctx, q, f, rec, payload)
	case *edgeRemoveFold:
		return s.applyFoldEdgeRemove(ctx, q, f, rec, payload)
	case *setFold:
		return s.applyFoldSet(ctx, q, f, rec, payload)
	case *skipFold:
		return nil
	case *multiInsertFold:
		return s.applyFoldMultiInsert(ctx, q, f, rec, payload)
	case *appendFold:
		return s.applyFoldAppend(ctx, q, f, rec, payload)
	case *vectorFold:
		return s.applyFoldVector(ctx, q, f, rec, payload)
	case *searchFold:
		return s.applyFoldSearch(ctx, q, f, rec, payload)
	case *spatialFold:
		return s.applyFoldSpatial(ctx, q, f, rec, payload)
	default:
		return fmt.Errorf("%w: %T", errUnknownFoldKind, fold)
	}
}

func (s *Store) applyFoldInsert(
	ctx context.Context,
	q queryMeta,
	fold *insertFold,
	rec record.Record,
	payload any,
) error {
	key, value := fold.invoke(rec, payload)
	col := q.QueryName()

	if mb, ok := q.QueryEngine().(MapBackend); ok {
		if err := mb.MapSet(ctx, col, key, value); err != nil {
			return fmt.Errorf("map set %s: %w", col, err)
		}

		s.notifyLive(q, col, key, value)

		return nil
	}

	return unsupportedEngine(errUnsupportedMapOps, q.QueryEngine().Profile().Name)
}

func (s *Store) applyFoldUpdate(
	ctx context.Context,
	q queryMeta,
	fold *updateFold,
	rec record.Record,
	payload any,
) error {
	key := fold.keyExtractor(payload)
	col := q.QueryName()

	if mu, ok := q.QueryEngine().(MapUpdater); ok {
		var updatedVal any

		if err := mu.MapUpdate(ctx, col, key, func(prev any) any {
			updatedVal = fold.invoke(rec, payload, prev)

			return updatedVal
		}); err != nil {
			return fmt.Errorf("map update %s: %w", col, err)
		}

		s.notifyLive(q, col, key, updatedVal)

		return nil
	}

	if mapBackend, ok := q.QueryEngine().(MapBackend); ok {
		prev, exists, err := mapBackend.MapGet(ctx, col, key)
		if err != nil {
			return fmt.Errorf("map get %s: %w", col, err)
		}

		var prevVal any
		if exists {
			prevVal = prev
		}

		updated := fold.invoke(rec, payload, prevVal)

		if err := mapBackend.MapSet(ctx, col, key, updated); err != nil {
			return fmt.Errorf("map set %s: %w", col, err)
		}

		s.notifyLive(q, col, key, updated)

		return nil
	}

	return unsupportedEngine(errUnsupportedMapOps, q.QueryEngine().Profile().Name)
}

func (s *Store) applyFoldRemove(
	ctx context.Context,
	q queryMeta,
	fold *removeFold,
	_ record.Record,
	payload any,
) error {
	key := fold.keyExtractor(payload)
	col := q.QueryName()

	if mb, ok := q.QueryEngine().(MapBackend); ok {
		if err := mb.MapDelete(ctx, col, key); err != nil {
			return fmt.Errorf("map delete %s: %w", col, err)
		}

		s.notifyLive(q, col, key, nil)

		return nil
	}

	return unsupportedEngine(errUnsupportedMapOps, q.QueryEngine().Profile().Name)
}

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
