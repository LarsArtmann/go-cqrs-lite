package metaengine

import (
	"context"
	"fmt"
	"reflect"
)

type queryRuntime struct {
	name          string
	adt           ADT
	readPattern   ReadPattern
	filters       []FieldPath
	sortField     string
	isPaginated   bool
	engine        Engine
	folds         []Fold
	foldByEvent   map[string]int
	inputTypeName string
}

type Store struct {
	engines     []Engine
	queries     map[string]queryRuntime
	byInputType map[string]string // Go input type name → query name
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

// Apply processes an event through ALL projections that listen to it.
func (s *Store) Apply(eventType string, payload any) error {
	for _, q := range s.queries {
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
	col := q.name

	switch fold.Kind {
	case FoldInsert:
		key, value := fold.callInsert(payload)
		if mb, ok := q.engine.(MapBackend); ok {
			return mb.MapSet(col, key, value)
		}
		return fmt.Errorf("engine %s does not support Map operations", q.engine.Profile().Name)

	case FoldUpdate:
		key := fold.callKey(payload)
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

	default:
		return fmt.Errorf("unknown fold kind: %s", fold.Kind)
	}
}

// Execute dispatches a query input to its assigned engine and returns the result.
func (s *Store) Execute(input any, opts ...ExecOption) (any, error) {
	return s.ExecuteCtx(context.Background(), input, opts...)
}

func (s *Store) ExecuteCtx(_ context.Context, input any, opts ...ExecOption) (any, error) {
	inputType := reflect.TypeOf(input).Name()
	queryName, ok := s.byInputType[inputType]
	if !ok {
		return nil, fmt.Errorf("no query declared for input type %q", inputType)
	}
	q := s.queries[queryName]
	return s.executeQuery(q, input, opts...)
}

func (s *Store) executeQuery(q queryRuntime, input any, opts ...ExecOption) (any, error) {
	cfg := applyExecOpts(opts)

	switch q.readPattern {
	case ReadPointLookup:
		key := extractKeyValue(input)
		if mb, ok := q.engine.(MapBackend); ok {
			val, ok, err := mb.MapGet(q.name, key)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, nil
			}
			return val, nil
		}
		return nil, fmt.Errorf("engine %s does not support Map reads", q.engine.Profile().Name)

	case ReadMembership:
		key := extractKeyValue(input)
		if sb, ok := q.engine.(SetBackend); ok {
			return sb.SetContains(q.name, key)
		}
		return nil, fmt.Errorf("engine %s does not support Set reads", q.engine.Profile().Name)

	case ReadFilteredScan:
		filterValues := extractFilterValues(input, q.filters)
		if sb, ok := q.engine.(ScanBackend); ok {
			results, err := sb.MapScan(q.name, q.filters, filterValues, q.sortField, cfg.limit)
			if err != nil {
				return nil, err
			}
			return results, nil // []any — caller wraps via ExecuteTyped
		}
		return nil, fmt.Errorf("engine %s does not support Scan reads", q.engine.Profile().Name)

	case ReadAggregate:
		if cb, ok := q.engine.(CounterBackend); ok {
			return cb.CounterGet(q.name)
		}
		return nil, fmt.Errorf("engine %s does not support Counter reads", q.engine.Profile().Name)

	case ReadTraversal:
		node := extractKeyValue(input)
		depth := extractDepthFromInput(input)
		if gb, ok := q.engine.(GraphBackend); ok {
			return gb.GraphNeighbors(q.name, node, depth)
		}
		return nil, fmt.Errorf("engine %s does not support Graph reads", q.engine.Profile().Name)

	default:
		return nil, fmt.Errorf("unsupported read pattern: %s", q.readPattern)
	}
}

// extractKeyValue gets the first field value from a query input struct.
func extractKeyValue(input any) any {
	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct || v.NumField() == 0 {
		return nil
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
	if v.Kind() == reflect.Ptr {
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
func reconstructPage[R any](raw any) R {
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

	slice := reflect.MakeSlice(reflect.SliceOf(elemType), 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		val := reflect.ValueOf(item)
		if val.Type().ConvertibleTo(elemType) {
			slice = reflect.Append(slice, val.Convert(elemType))
		}
	}

	result := reflect.New(t).Elem()
	result.FieldByName("Items").Set(slice)
	return result.Interface().(R)
}

// ExecuteTyped is the type-safe wrapper. It dispatches the query and type-asserts the result.
//
// For paginated queries (result type is Page[T]), it reconstructs the typed slice
// from the []any returned by the engine.
//
// Usage: result, err := metaengine.ExecuteTyped[FindUser, FindUserResult](ctx, store, FindUser{ID: uid})
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
	// If R is Page[T], raw is []any — reconstruct typed page.
	if isPageType[R]() {
		return reconstructPage[R](raw), nil
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
