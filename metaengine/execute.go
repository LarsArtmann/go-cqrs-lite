package metaengine

import (
	"context"
	"fmt"
	"reflect"
)

// Execute dispatches a query input to its query's engine and returns the result.
func (s *Store) Execute(input any) (any, error) {
	return s.ExecuteCtx(context.Background(), input)
}

func (s *Store) ExecuteCtx(ctx context.Context, input any) (any, error) {
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

	return s.executeQuery(ctx, q, input)
}

func (s *Store) executeQuery(
	ctx context.Context,
	q queryRuntime,
	input any,
) (any, error) {
	switch q.readPattern {
	case ReadPointLookup:
		key := extractKeyValueByType(input, q.keyType)
		if mb, ok := q.engine.(MapBackend); ok {
			val, ok, err := mb.MapGet(ctx, q.name, key)
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
		key := extractKeyValueByType(input, q.keyType)
		if sb, ok := q.engine.(SetBackend); ok {
			return sb.SetContains(ctx, q.name, key)
		}

		return nil, fmt.Errorf("engine %s does not support Set reads", q.engine.Profile().Name)

	case ReadFilteredScan:
		return s.executeFilteredScan(ctx, q, input)

	case ReadAggregate:
		if cb, ok := q.engine.(CounterBackend); ok {
			return cb.CounterGet(ctx, q.name)
		}

		return nil, fmt.Errorf("engine %s does not support Counter reads", q.engine.Profile().Name)

	case ReadTraversal:
		node := extractKeyValueByType(input, q.keyType)
		if node == nil {
			node = extractFirstDomainField(input)
		}

		depth := extractDepthFromInput(input)
		if gb, ok := q.engine.(GraphBackend); ok {
			return gb.GraphNeighbors(ctx, q.name, node, depth)
		}

		return nil, fmt.Errorf("engine %s does not support Graph reads", q.engine.Profile().Name)

	case ReadMultiLookup:
		key := extractFirstDomainField(input)
		if mb, ok := q.engine.(MultimapBackend); ok {
			return mb.MultiGet(ctx, q.name, key)
		}

		return nil, fmt.Errorf("engine %s does not support Multimap reads", q.engine.Profile().Name)

	case ReadLogTail:
		limit := extractLimitFromInput(input)
		if lb, ok := q.engine.(LogBackend); ok {
			return lb.LogTail(ctx, q.name, limit)
		}

		return nil, fmt.Errorf("engine %s does not support Log reads", q.engine.Profile().Name)

	default:
		return nil, fmt.Errorf("unsupported read pattern: %s", q.readPattern)
	}
}

func (s *Store) executeFilteredScan(ctx context.Context, q queryRuntime, input any) (any, error) {
	limit := extractLimitFromInput(input)
	if limit == 0 {
		limit = 100
	}

	cursor := extractCursorFromInput(input)

	var cursorVal any
	if cursor != nil {
		cursorVal = cursor.Value
	}

	// Build filter predicates from typed closures.
	filterPredicates, err := buildFilterPredicates(q, input)
	if err != nil {
		return nil, err
	}

	// Determine sort accessor.
	var sortFunc func(a, b any) int
	if q.config.sortAccessor.closure != nil {
		sortFunc = buildSortFunc(q.config.sortAccessor.closure)
	} else {
		// No explicit SortOn — try default: sort by first time.Time field.
		sortFunc = nil // engine decides internally
	}

	if sb, ok := q.engine.(ScanBackend); ok {
		return sb.MapScan(ctx, q.name, filterPredicates, sortFunc, cursorVal, limit)
	}

	return nil, fmt.Errorf("engine %s does not support Scan reads", q.engine.Profile().Name)
}

// buildFilterPredicates creates runtime filter predicates from typed closures
// declared via FilterOn. Each closure extracts a comparable value from a result
// item. The matching filter value is extracted from the query input by type.
func buildFilterPredicates(q queryRuntime, input any) ([]filterPredicate, error) {
	if len(q.config.filterAccessors) == 0 {
		return nil, nil
	}

	var predicates []filterPredicate

	for _, acc := range q.config.filterAccessors {
		// Extract the expected filter value from the input by type matching.
		expected := extractValueByType(input, acc.returnType)
		if expected == nil {
			// No matching field in input — skip this filter (no constraint).
			continue
		}

		closure := acc.closure

		predicates = append(predicates, filterPredicate{
			expected: expected,
			test: func(item any) bool {
				rv := reflect.ValueOf(closure)
				result := rv.Call([]reflect.Value{reflect.ValueOf(item)})

				return reflect.DeepEqual(result[0].Interface(), expected)
			},
		})
	}

	return predicates, nil
}

// extractValueByType finds a field in the input struct whose type matches
// the given type, and returns its value. Returns nil if not found or ambiguous.
func extractValueByType(input any, targetType reflect.Type) any {
	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()

	metaNames := map[string]bool{"Limit": true, "After": true, "Depth": true}

	foundIdx := -1

	for i := range t.NumField() {
		if !t.Field(i).IsExported() || metaNames[t.Field(i).Name] {
			continue
		}

		if t.Field(i).Type == targetType {
			if foundIdx >= 0 {
				return nil // ambiguous
			}

			foundIdx = i
		}
	}

	if foundIdx < 0 {
		return nil
	}

	return v.Field(foundIdx).Interface()
}

// buildSortFunc creates a comparator from a SortOn closure.
// The closure extracts a sort key from each result item.
// When comparing against a cursor value (which is already a raw sort key,
// not a result item), the closure is skipped and the value is used directly.
func buildSortFunc(closure any) func(a, b any) int {
	rv := reflect.ValueOf(closure)
	paramType := rv.Type().In(0)

	extractKey := func(item any) any {
		if reflect.TypeOf(item) == paramType {
			return rv.Call([]reflect.Value{reflect.ValueOf(item)})[0].Interface()
		}

		return item
	}

	return func(a, b any) int {
		return compareValue(extractKey(a), extractKey(b))
	}
}

// filterPredicate is a runtime filter test against a result item.
type filterPredicate struct {
	expected any
	test     func(item any) bool
}

func (s *Store) sortKeyFn(inputType string) func(any) any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	queryName, ok := s.byInputType[inputType]
	if !ok {
		return nil
	}

	q := s.queries[queryName]
	if q.config.sortAccessor.closure == nil {
		return nil
	}

	rv := reflect.ValueOf(q.config.sortAccessor.closure)

	return func(item any) any {
		return rv.Call([]reflect.Value{reflect.ValueOf(item)})[0].Interface()
	}
}

// ExecuteTyped is the type-safe wrapper. It dispatches the query and type-asserts
// the result.
//
// For collection results (structs with a []T field), it reconstructs the typed
// slice from the []any returned by the engine.
//
// Usage: result, err := metaengine.ExecuteTyped[FindUser, FindUserResult](ctx, store, FindUser{ID: uid}).
func ExecuteTyped[Q any, R any](
	_ context.Context,
	store *Store,
	input Q,
) (R, error) {
	var zero R

	raw, err := store.Execute(input)
	if err != nil {
		return zero, err
	}

	if raw == nil {
		return zero, nil
	}

	if isCollectionResult[R]() {
		limit := extractLimitFromInput(input)
		sortFn := store.sortKeyFn(qualifiedTypeName(input))

		return reconstructCollection[R](raw, limit, sortFn), nil
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
