package metaengine

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

type queryRuntime struct {
	name          string
	adt           ADT
	engine        Engine
	complexity    Complexity
	folds         []Fold
	foldByEvent   map[string]int
	readPattern   ReadPattern
	config        QueryConfig
	keyType       reflect.Type
	resultType    reflect.Type
	inputTypeName string
}

type Store struct {
	mu           sync.RWMutex
	engines      []Engine
	queries      map[string]queryRuntime
	byInputType  map[string]string
	plan         *PlanResult
	poisoned     sync.Map // collection name → poison error
	appliedEvent sync.Map // event ID → struct{} (idempotent Apply dedup)
	hooks        *Hooks   // observability hooks (nil = no-op)
	eventLog     *EventLog
	queryDecls   []any          // original query declarations (for Verify)
	coalescer    *ReadCoalescer // optional read coalescer (nil = disabled)
	writeCount   atomic.Int64   // total events applied (for WorkloadStats)
	readCount    atomic.Int64   // total queries executed (for WorkloadStats)
	startTime    time.Time      // when the store was created (for rate calculation)
	watcherMu    sync.Mutex
	watchers     map[string][]*watcherEntry // collection → watcher entries
	replays      map[string]replayRecorder  // collection → replay recorder (nil = no replay)
}

func (s *Store) Plan() *PlanResult { return s.plan }

// CollectionInfo describes a planned query collection.
type CollectionInfo struct {
	Name        string
	ADT         ADT
	ReadPattern ReadPattern
	EngineName  string
	Complexity  Complexity
}

// Collections returns metadata for every registered query collection.
// The result is sorted by collection name.
func (s *Store) Collections() []CollectionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]CollectionInfo, 0, len(s.queries))

	for _, name := range slices.Sorted(maps.Keys(s.queries)) {
		q := s.queries[name]
		result = append(result, CollectionInfo{
			Name:        q.name,
			ADT:         q.adt,
			ReadPattern: q.readPattern,
			EngineName:  q.engine.Profile().Name,
			Complexity:  q.complexity,
		})
	}

	return result
}

// collectionEngine returns the engine assigned to a query/collection by name.
// Used by TypedReader to access the engine for typed reads without going through
// the reflective Execute path.
func (s *Store) collectionEngine(collection string) (Engine, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	q, ok := s.queries[collection]
	if !ok {
		return nil, false
	}

	return q.engine, true
}

// IsPoisoned returns the poison error if the collection was poisoned by a fold
// panic, or nil if healthy. Once poisoned, a collection refuses reads until
// the store is recreated (or the poison is cleared via Reset).
func (s *Store) IsPoisoned(collection string) error {
	if v, ok := s.poisoned.Load(collection); ok {
		return v.(error)
	}

	return nil
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
		for et := range q.foldByEvent {
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
	s.watcherMu.Lock()
	defer s.watcherMu.Unlock()

	if s.watchers == nil {
		s.watchers = make(map[string][]*watcherEntry)
	}

	s.watchers[collection] = append(s.watchers[collection], entry)
}

// registerReplay attaches a replay recorder to a collection so that
// notifyWatchers records each value change with a monotonic sequence number.
// Called by Watcher.WithReplay. Thread-safe.
func (s *Store) registerReplay(collection string, recorder replayRecorder) {
	s.watcherMu.Lock()
	defer s.watcherMu.Unlock()

	if s.replays == nil {
		s.replays = make(map[string]replayRecorder)
	}

	s.replays[collection] = recorder
}

// unregisterReplay removes the replay recorder for a collection.
// Called by Watcher.Close. Thread-safe.
func (s *Store) unregisterReplay(collection string) {
	s.watcherMu.Lock()
	defer s.watcherMu.Unlock()

	if s.replays != nil {
		delete(s.replays, collection)
	}
}

// keysMatch compares two keys for watcher filtering. Uses reflect.DeepEqual
// for correctness across types (strings, ints, branded IDs).
func keysMatch(a, b any) bool {
	return reflect.DeepEqual(a, b)
}

// notifyWatchers sends a value update to all watchers of a collection that
// match the given key. A watcher with a nil key receives all notifications;
// a watcher with a non-nil key receives only notifications for that key.
// Non-blocking: if a watcher's channel is full, the notification is dropped.
//
// When a replay recorder is registered for the collection, the value is
// recorded with a monotonic sequence number and sent as a watcherNotification
// wrapper so WatchWithSeq can recover the seq. The Watch adapter unwraps
// watcherNotification transparently, so existing Watch callers are unaffected.
func (s *Store) notifyWatchers(collection string, key any, value any) {
	s.watcherMu.Lock()
	entries := s.watchers[collection]
	recorder := s.replays[collection]
	s.watcherMu.Unlock()

	var notif watcherNotification
	if recorder != nil {
		notif = watcherNotification{seq: recorder.recordValue(value), value: value}
	}

	for _, entry := range entries {
		if entry.closed {
			continue
		}

		// Per-key filter: if the watcher subscribed to a specific key,
		// only forward notifications for that key.
		if entry.key != nil && !keysMatch(entry.key, key) {
			continue
		}

		if recorder != nil {
			select {
			case entry.ch <- notif:
			default: // drop if consumer is slow
			}
		} else {
			select {
			case entry.ch <- value:
			default: // drop if consumer is slow
			}
		}
	}
}

// Apply processes an event through ALL queries that listen to it.
// Each query has its own independent projection — the same event updates
// each matching query's collection separately.
func (s *Store) Apply(ctx context.Context, eventType string, payload any) error {
	s.writeCount.Add(1)

	if s.eventLog != nil {
		s.eventLog.Record(eventType, payload)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, name := range slices.Sorted(maps.Keys(s.queries)) {
		q := s.queries[name]

		foldIdx, ok := q.foldByEvent[eventType]
		if !ok {
			continue
		}

		fold := q.folds[foldIdx]
		if err := s.applyFold(ctx, q, fold, payload); err != nil {
			return fmt.Errorf("query %q fold for %s: %w", q.name, eventType, err)
		}
	}

	return nil
}

// EventInput pairs an event type with its payload for batch application.
type EventInput struct {
	Type    string
	Payload any
}

// ApplyBatch processes multiple events through all queries in one call.
// Events are applied sequentially; on the first error, remaining events are
// skipped and the error is returned. This is the primary API for replay
// scenarios where many events need to be processed.
func (s *Store) ApplyBatch(ctx context.Context, events []EventInput) error {
	for _, evt := range events {
		if err := s.Apply(ctx, evt.Type, evt.Payload); err != nil {
			return fmt.Errorf("batch apply event %q: %w", evt.Type, err)
		}
	}

	return nil
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

	if _, exists := s.appliedEvent.LoadOrStore(eventID, struct{}{}); exists {
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

func (s *Store) applyFold(ctx context.Context, q queryRuntime, fold Fold, payload any) (err error) {
	start := time.Now()

	defer func() {
		if s.hooks != nil && s.hooks.OnFold != nil {
			s.hooks.OnFold(q.name, fold.EventType, fold.Kind, time.Since(start), err)
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			poisonErr := fmt.Errorf("%w: collection %q, panic: %v", ErrPoisoned, q.name, r)
			s.poisoned.Store(q.name, poisonErr)
			err = poisonErr
		}
	}()

	switch fold.Kind {
	case FoldInsert:
		return s.applyFoldInsert(ctx, q, fold, payload)
	case FoldUpdate:
		return s.applyFoldUpdate(ctx, q, fold, payload)
	case FoldRemove:
		return s.applyFoldRemove(ctx, q, fold, payload)
	case FoldCount:
		return s.applyFoldCount(ctx, q, fold, payload)
	case FoldEdge:
		return s.applyFoldEdge(ctx, q, fold, payload)
	case FoldSet:
		return s.applyFoldSet(ctx, q, fold, payload)
	case FoldSkip:
		return nil
	case FoldMultiInsert:
		return s.applyFoldMultiInsert(ctx, q, fold, payload)
	case FoldAppend:
		return s.applyFoldAppend(ctx, q, fold, payload)
	case FoldVector:
		return s.applyFoldVector(ctx, q, fold, payload)
	case FoldSearch:
		return s.applyFoldSearch(ctx, q, fold, payload)
	default:
		return fmt.Errorf("%w: %s", errUnknownFoldKind, fold.Kind)
	}
}

func (s *Store) applyFoldInsert(ctx context.Context, q queryRuntime, fold Fold, payload any) error {
	key, value := fold.callInsert(payload)
	col := q.name

	if mb, ok := q.engine.(MapBackend); ok {
		if err := mb.MapSet(ctx, col, key, value); err != nil {
			return fmt.Errorf("map set %s: %w", col, err)
		}

		s.notifyWatchers(col, key, value)

		return nil
	}

	return unsupportedEngine(errUnsupportedMapOps, q.engine.Profile().Name)
}

func (s *Store) applyFoldUpdate(ctx context.Context, q queryRuntime, fold Fold, payload any) error {
	key := fold.callKey(payload)
	col := q.name

	if mu, ok := q.engine.(MapUpdater); ok {
		var updatedVal any

		if err := mu.MapUpdate(ctx, col, key, func(prev any) any {
			updatedVal = fold.callUpdate(payload, prev)

			return updatedVal
		}); err != nil {
			return fmt.Errorf("map update %s: %w", col, err)
		}

		s.notifyWatchers(col, key, updatedVal)

		return nil
	}

	if mapBackend, ok := q.engine.(MapBackend); ok {
		prev, exists, err := mapBackend.MapGet(ctx, col, key)
		if err != nil {
			return fmt.Errorf("map get %s: %w", col, err)
		}

		var prevVal any
		if exists {
			prevVal = prev
		}

		updated := fold.callUpdate(payload, prevVal)

		if err := mapBackend.MapSet(ctx, col, key, updated); err != nil {
			return fmt.Errorf("map set %s: %w", col, err)
		}

		s.notifyWatchers(col, key, updated)

		return nil
	}

	return unsupportedEngine(errUnsupportedMapOps, q.engine.Profile().Name)
}

func (s *Store) applyFoldRemove(ctx context.Context, q queryRuntime, fold Fold, payload any) error {
	key := fold.callKey(payload)
	col := q.name

	if mb, ok := q.engine.(MapBackend); ok {
		if err := mb.MapDelete(ctx, col, key); err != nil {
			return fmt.Errorf("map delete %s: %w", col, err)
		}

		s.notifyWatchers(col, key, nil)

		return nil
	}

	return unsupportedEngine(errUnsupportedMapOps, q.engine.Profile().Name)
}

func (s *Store) applyFoldCount(ctx context.Context, q queryRuntime, fold Fold, payload any) error {
	col := q.name
	delta := fold.callCount(payload)

	if cb, ok := q.engine.(CounterBackend); ok {
		if err := cb.CounterIncrement(ctx, col, delta); err != nil {
			return fmt.Errorf("counter increment %s: %w", col, err)
		}

		return nil
	}

	return unsupportedEngine(errUnsupportedCounterOps, q.engine.Profile().Name)
}

func (s *Store) applyFoldEdge(ctx context.Context, q queryRuntime, fold Fold, payload any) error {
	col := q.name
	edge := fold.callEdge(payload)

	if gb, ok := q.engine.(GraphBackend); ok {
		if err := gb.GraphAddEdge(ctx, col, edge); err != nil {
			return fmt.Errorf("graph add edge %s: %w", col, err)
		}

		return nil
	}

	return unsupportedEngine(errUnsupportedGraphOps, q.engine.Profile().Name)
}

func (s *Store) applyFoldSet(ctx context.Context, q queryRuntime, fold Fold, payload any) error {
	col := q.name
	key := fold.callSet(payload)

	if sb, ok := q.engine.(SetBackend); ok {
		if err := sb.SetAdd(ctx, col, key); err != nil {
			return fmt.Errorf("set add %s: %w", col, err)
		}

		return nil
	}

	return unsupportedEngine(errUnsupportedSetOps, q.engine.Profile().Name)
}

func (s *Store) applyFoldMultiInsert(
	ctx context.Context,
	q queryRuntime,
	fold Fold,
	payload any,
) error {
	col := q.name
	entry := fold.callMultiInsert(payload)

	if mb, ok := q.engine.(MultimapBackend); ok {
		if err := mb.MultiAdd(ctx, col, entry.Key, entry.Value); err != nil {
			return fmt.Errorf("multi add %s: %w", col, err)
		}

		return nil
	}

	return unsupportedEngine(errUnsupportedMultimapOps, q.engine.Profile().Name)
}

func (s *Store) applyFoldAppend(ctx context.Context, q queryRuntime, fold Fold, payload any) error {
	col := q.name
	app := fold.callAppend(payload)

	if lb, ok := q.engine.(LogBackend); ok {
		if err := lb.LogAppend(ctx, col, app.Value); err != nil {
			return fmt.Errorf("log append %s: %w", col, err)
		}

		return nil
	}

	return unsupportedEngine(errUnsupportedLogOps, q.engine.Profile().Name)
}

func (s *Store) applyFoldVector(ctx context.Context, q queryRuntime, fold Fold, payload any) error {
	col := q.name
	emb := fold.callVector(payload)

	if vb, ok := q.engine.(VectorBackend); ok {
		if err := vb.VectorInsert(ctx, col, emb); err != nil {
			return fmt.Errorf("vector insert %s: %w", col, err)
		}

		return nil
	}

	return unsupportedEngine(errUnsupportedVectorOps, q.engine.Profile().Name)
}

func (s *Store) applyFoldSearch(ctx context.Context, q queryRuntime, fold Fold, payload any) error {
	col := q.name
	doc := fold.callSearch(payload)

	if sb, ok := q.engine.(SearchBackend); ok {
		if err := sb.SearchInsert(ctx, col, doc); err != nil {
			return fmt.Errorf("search insert %s: %w", col, err)
		}

		return nil
	}

	return unsupportedEngine(errUnsupportedSearchOps, q.engine.Profile().Name)
}
