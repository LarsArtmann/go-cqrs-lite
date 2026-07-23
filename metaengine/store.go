package metaengine

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

type modelRuntime struct {
	name        string
	adt         ADT
	engine      Engine
	complexity  Complexity
	folds       []Fold
	foldByEvent map[string]int
}

type queryRuntime struct {
	name          string
	modelName     string
	readPattern   ReadPattern
	filters       []FieldPath
	sortField     string
	isPaginated   bool
	inputTypeName string
}

type Store struct {
	mu          sync.RWMutex
	engines     []Engine
	models      map[string]modelRuntime
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

// Apply processes an event through ALL models that listen to it.
// One event updates each matching model exactly once, regardless of how many
// queries read from that model.
func (s *Store) Apply(eventType string, payload any) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, m := range s.models {
		foldIdx, ok := m.foldByEvent[eventType]
		if !ok {
			continue
		}

		fold := m.folds[foldIdx]
		if err := s.applyFold(m, fold, payload); err != nil {
			return fmt.Errorf("model %q fold for %s: %w", m.name, eventType, err)
		}
	}

	return nil
}

func (s *Store) applyFold(m modelRuntime, fold Fold, payload any) error {
	col := m.name

	switch fold.Kind {
	case FoldInsert:
		key, value := fold.callInsert(payload)
		if mb, ok := m.engine.(MapBackend); ok {
			return mb.MapSet(col, key, value)
		}

		return fmt.Errorf("engine %s does not support Map operations", m.engine.Profile().Name)

	case FoldUpdate:
		key := fold.callKey(payload)
		if mb, ok := m.engine.(MapBackend); ok {
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

		return fmt.Errorf("engine %s does not support Map operations", m.engine.Profile().Name)

	case FoldRemove:
		key := fold.callKey(payload)
		if mb, ok := m.engine.(MapBackend); ok {
			return mb.MapDelete(col, key)
		}

		return fmt.Errorf("engine %s does not support Map operations", m.engine.Profile().Name)

	case FoldCount:
		delta := fold.callCount(payload)
		if cb, ok := m.engine.(CounterBackend); ok {
			return cb.CounterIncrement(col, delta)
		}

		return fmt.Errorf("engine %s does not support Counter operations", m.engine.Profile().Name)

	case FoldEdge:
		edge := fold.callEdge(payload)
		if gb, ok := m.engine.(GraphBackend); ok {
			return gb.GraphAddEdge(col, edge)
		}

		return fmt.Errorf("engine %s does not support Graph operations", m.engine.Profile().Name)

	case FoldSet:
		key := fold.callSet(payload)
		if sb, ok := m.engine.(SetBackend); ok {
			return sb.SetAdd(col, key)
		}

		return fmt.Errorf("engine %s does not support Set operations", m.engine.Profile().Name)

	case FoldSkip:
		return nil

	default:
		return fmt.Errorf("unknown fold kind: %s", fold.Kind)
	}
}

// Execute dispatches a query input to its model's engine and returns the result.
func (s *Store) Execute(input any, opts ...ExecOption) (any, error) {
	return s.ExecuteCtx(context.Background(), input, opts...)
}

func (s *Store) ExecuteCtx(ctx context.Context, input any, opts ...ExecOption) (any, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	inputType := qualifiedTypeName(input)

	queryName, ok := s.byInputType[inputType]
	if !ok {
		return nil, fmt.Errorf("no query declared for input type %q", inputType)
	}

	q := s.queries[queryName]

	return s.executeQuery(ctx, q, input, opts...)
}

func (s *Store) executeQuery(
	_ context.Context,
	q queryRuntime,
	input any,
	opts ...ExecOption,
) (any, error) {
	cfg := applyExecOpts(opts)
	m := s.models[q.modelName]

	switch q.readPattern {
	case ReadPointLookup:
		key := extractKeyValue(input)
		if mb, ok := m.engine.(MapBackend); ok {
			val, ok, err := mb.MapGet(m.name, key)
			if err != nil {
				return nil, err
			}

			if !ok {
				return nil, nil
			}

			return val, nil
		}

		return nil, fmt.Errorf("engine %s does not support Map reads", m.engine.Profile().Name)

	case ReadMembership:
		key := extractKeyValue(input)
		if sb, ok := m.engine.(SetBackend); ok {
			return sb.SetContains(m.name, key)
		}

		return nil, fmt.Errorf("engine %s does not support Set reads", m.engine.Profile().Name)

	case ReadFilteredScan:
		filterValues := extractFilterValues(input, q.filters)
		if sb, ok := m.engine.(ScanBackend); ok {
			results, err := sb.MapScan(m.name, q.filters, filterValues, q.sortField, cfg.limit)
			if err != nil {
				return nil, err
			}

			return results, nil
		}

		return nil, fmt.Errorf("engine %s does not support Scan reads", m.engine.Profile().Name)

	case ReadAggregate:
		if cb, ok := m.engine.(CounterBackend); ok {
			return cb.CounterGet(m.name)
		}

		return nil, fmt.Errorf("engine %s does not support Counter reads", m.engine.Profile().Name)

	case ReadTraversal:
		node := extractKeyValue(input)

		depth := extractDepthFromInput(input)
		if gb, ok := m.engine.(GraphBackend); ok {
			return gb.GraphNeighbors(m.name, node, depth)
		}

		return nil, fmt.Errorf("engine %s does not support Graph reads", m.engine.Profile().Name)

	default:
		return nil, fmt.Errorf("unsupported read pattern: %s", q.readPattern)
	}
}

// extractKeyValue gets the key value from a query input struct.
// Priority: field tagged metaengine:"key" > first exported field.
func extractKeyValue(input any) any {
	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct || v.NumField() == 0 {
		return nil
	}

	t := v.Type()
	for i := range t.NumField() {
		if t.Field(i).Tag.Get("metaengine") == "key" {
			return v.Field(i).Interface()
		}
	}

	return v.Field(0).Interface()
}

func extractFilterValues(input any, filters []FieldPath) map[string]any {
	values := make(map[string]any, len(filters))
	for _, f := range filters {
		values[f.Field] = getFieldValue(input, f.Field)
	}

	return values
}

func extractDepthFromInput(input any) int {
	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return 1
	}

	f := v.FieldByName("Depth")
	if !f.IsValid() || f.Kind() != reflect.Int {
		return 1
	}

	return int(f.Int())
}

// isPageType checks via reflection if R is a Page[T] by field shape.
func isPageType[R any]() bool {
	var zero R

	_, ok := unwrapPageType(reflect.TypeOf(zero))

	return ok
}

// reconstructPage builds a typed Page[T] from a []any returned by the engine.
// If the engine returned more items than the limit, HasMore is set to true
// and Next is populated with a cursor pointing past the current page.
func reconstructPage[R any](raw any, limit int) R {
	var zero R

	t := reflect.TypeOf(zero)

	elemType, ok := unwrapPageType(t)
	if !ok {
		return zero
	}

	items, ok := raw.([]any)
	if !ok {
		return zero
	}

	hasMore := limit > 0 && len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	slice := reflect.MakeSlice(reflect.SliceOf(elemType), 0, len(items))

	var lastItem any

	for _, item := range items {
		if item == nil {
			continue
		}

		val := reflect.ValueOf(item)
		if val.Type().ConvertibleTo(elemType) {
			slice = reflect.Append(slice, val.Convert(elemType))
			lastItem = item
		}
	}

	result := reflect.New(t).Elem()
	result.FieldByName("Items").Set(slice)

	if hasMore {
		result.FieldByName("HasMore").SetBool(true)

		if lastItem != nil {
			cursor := &Cursor{Value: lastItem}

			cursorField := result.FieldByName("Next")
			if cursorField.IsValid() && cursorField.Kind() == reflect.Pointer {
				cursorField.Set(reflect.ValueOf(cursor))
			}
		}
	}

	return result.Interface().(R)
}

// ExecuteTyped is the type-safe wrapper. It dispatches the query and type-asserts the result.
//
// For paginated queries (result type is Page[T]), it reconstructs the typed slice
// from the []any returned by the engine.
//
// Usage: result, err := metaengine.ExecuteTyped[FindUser, FindUserResult](ctx, store, FindUser{ID: uid}).
func ExecuteTyped[Q any, R any](
	_ context.Context,
	store *Store,
	input Q,
	opts ...ExecOption,
) (R, error) {
	var zero R

	raw, err := store.Execute(input, opts...)
	if err != nil {
		return zero, err
	}

	if raw == nil {
		return zero, nil
	}

	if isPageType[R]() {
		cfg := applyExecOpts(opts)

		return reconstructPage[R](raw, cfg.limit), nil
	}

	result, ok := raw.(R)
	if !ok {
		return zero, fmt.Errorf(
			"metaengine.ExecuteTyped: result type %T does not match expected %T",
			raw, zero,
		)
	}

	return result, nil
}
