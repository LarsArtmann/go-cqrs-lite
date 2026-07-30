package metaengine

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"sync"
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
	inputTypeName string
}

type Store struct {
	mu          sync.RWMutex
	engines     []Engine
	queries     map[string]queryRuntime
	byInputType map[string]string
	plan        *PlanResult
}

func (s *Store) Plan() *PlanResult { return s.plan }

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

// Apply processes an event through ALL queries that listen to it.
// Each query has its own independent projection — the same event updates
// each matching query's collection separately.
func (s *Store) Apply(ctx context.Context, eventType string, payload any) error {
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

func (s *Store) applyFold(ctx context.Context, q queryRuntime, fold Fold, payload any) error {
	col := q.name

	switch fold.Kind {
	case FoldInsert:
		key, value := fold.callInsert(payload)
		if mb, ok := q.engine.(MapBackend); ok {
			if err := mb.MapSet(ctx, col, key, value); err != nil {
				return fmt.Errorf("map set %s: %w", col, err)
			}

			return nil
		}

		return unsupportedEngine(errUnsupportedMapOps, q.engine.Profile().Name)

	case FoldUpdate:
		key := fold.callKey(payload)

		if mu, ok := q.engine.(MapUpdater); ok {
			if err := mu.MapUpdate(ctx, col, key, func(prev any) any {
				return fold.callUpdate(payload, prev)
			}); err != nil {
				return fmt.Errorf("map update %s: %w", col, err)
			}

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

			return nil
		}

		return unsupportedEngine(errUnsupportedMapOps, q.engine.Profile().Name)

	case FoldRemove:
		key := fold.callKey(payload)
		if mb, ok := q.engine.(MapBackend); ok {
			if err := mb.MapDelete(ctx, col, key); err != nil {
				return fmt.Errorf("map delete %s: %w", col, err)
			}

			return nil
		}

		return unsupportedEngine(errUnsupportedMapOps, q.engine.Profile().Name)

	case FoldCount:
		delta := fold.callCount(payload)
		if cb, ok := q.engine.(CounterBackend); ok {
			if err := cb.CounterIncrement(ctx, col, delta); err != nil {
				return fmt.Errorf("counter increment %s: %w", col, err)
			}

			return nil
		}

		return unsupportedEngine(errUnsupportedCounterOps, q.engine.Profile().Name)

	case FoldEdge:
		edge := fold.callEdge(payload)
		if gb, ok := q.engine.(GraphBackend); ok {
			if err := gb.GraphAddEdge(ctx, col, edge); err != nil {
				return fmt.Errorf("graph add edge %s: %w", col, err)
			}

			return nil
		}

		return unsupportedEngine(errUnsupportedGraphOps, q.engine.Profile().Name)

	case FoldSet:
		key := fold.callSet(payload)
		if sb, ok := q.engine.(SetBackend); ok {
			if err := sb.SetAdd(ctx, col, key); err != nil {
				return fmt.Errorf("set add %s: %w", col, err)
			}

			return nil
		}

		return unsupportedEngine(errUnsupportedSetOps, q.engine.Profile().Name)

	case FoldSkip:
		return nil

	case FoldMultiInsert:
		entry := fold.callMultiInsert(payload)
		if mb, ok := q.engine.(MultimapBackend); ok {
			if err := mb.MultiAdd(ctx, col, entry.Key, entry.Value); err != nil {
				return fmt.Errorf("multi add %s: %w", col, err)
			}

			return nil
		}

		return unsupportedEngine(errUnsupportedMultimapOps, q.engine.Profile().Name)

	case FoldAppend:
		app := fold.callAppend(payload)
		if lb, ok := q.engine.(LogBackend); ok {
			if err := lb.LogAppend(ctx, col, app.Value); err != nil {
				return fmt.Errorf("log append %s: %w", col, err)
			}

			return nil
		}

		return unsupportedEngine(errUnsupportedLogOps, q.engine.Profile().Name)

	default:
		return fmt.Errorf("%w: %s", errUnknownFoldKind, fold.Kind)
	}
}
