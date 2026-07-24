package metaengine

import (
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
func (s *Store) Apply(eventType string, payload any) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, name := range slices.Sorted(maps.Keys(s.queries)) {
		q := s.queries[name]
		foldIdx, ok := q.foldByEvent[eventType]
		if !ok {
			continue
		}

		fold := q.folds[foldIdx]
		if err := s.applyFold(q, fold, payload); err != nil {
			return fmt.Errorf("query %q fold for %s: %w", q.name, eventType, err)
		}
	}

	return nil
}

func (s *Store) applyFold(q queryRuntime, fold Fold, payload any) error {
	col := q.name // each query's collection is named after the query

	switch fold.Kind {
	case FoldInsert:
		key, value := fold.callInsert(payload)
		if mb, ok := q.engine.(MapBackend); ok {
			return mb.MapSet(col, key, value)
		}

		return fmt.Errorf("engine %s does not support Map operations", q.engine.Profile().Name)

	case FoldUpdate:
		key := fold.callKey(payload)

		// Fast path: engine supports atomic read-modify-write.
		if mu, ok := q.engine.(MapUpdater); ok {
			return mu.MapUpdate(col, key, func(prev any) any {
				return fold.callUpdate(payload, prev)
			})
		}

		// Fallback: non-atomic read-modify-write (may lose updates under concurrency).
		if mb, ok := q.engine.(MapBackend); ok {
			prev, exists, err := mb.MapGet(col, key)
			if err != nil {
				return err
			}

			var prevVal any
			if exists {
				prevVal = prev
			}

			updated := fold.callUpdate(payload, prevVal)

			return mb.MapSet(col, key, updated)
		}

		return fmt.Errorf("engine %s does not support Map operations", q.engine.Profile().Name)

	case FoldRemove:
		key := fold.callKey(payload)
		if mb, ok := q.engine.(MapBackend); ok {
			return mb.MapDelete(col, key)
		}

		return fmt.Errorf("engine %s does not support Map operations", q.engine.Profile().Name)

	case FoldCount:
		delta := fold.callCount(payload)
		if cb, ok := q.engine.(CounterBackend); ok {
			return cb.CounterIncrement(col, delta)
		}

		return fmt.Errorf("engine %s does not support Counter operations", q.engine.Profile().Name)

	case FoldEdge:
		edge := fold.callEdge(payload)
		if gb, ok := q.engine.(GraphBackend); ok {
			return gb.GraphAddEdge(col, edge)
		}

		return fmt.Errorf("engine %s does not support Graph operations", q.engine.Profile().Name)

	case FoldSet:
		key := fold.callSet(payload)
		if sb, ok := q.engine.(SetBackend); ok {
			return sb.SetAdd(col, key)
		}

		return fmt.Errorf("engine %s does not support Set operations", q.engine.Profile().Name)

	case FoldSkip:
		return nil

	case FoldMultiInsert:
		entry := fold.callMultiInsert(payload)
		if mb, ok := q.engine.(MultimapBackend); ok {
			return mb.MultiAdd(col, entry.Key, entry.Value)
		}

		return fmt.Errorf("engine %s does not support Multimap operations", q.engine.Profile().Name)

	case FoldAppend:
		app := fold.callAppend(payload)
		if lb, ok := q.engine.(LogBackend); ok {
			return lb.LogAppend(col, app.Value)
		}

		return fmt.Errorf("engine %s does not support Log operations", q.engine.Profile().Name)

	default:
		return fmt.Errorf("unknown fold kind: %s", fold.Kind)
	}
}
