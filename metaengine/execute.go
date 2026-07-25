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
		return nil, fmt.Errorf("metaengine.ExecuteCtx: %w", ctx.Err())
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	inputType := qualifiedTypeName(input)

	queryName, ok := s.byInputType[inputType]
	if !ok {
		return nil, fmt.Errorf("%w: %q", errNoQueryForInputType, inputType)
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
			val, found, err := mb.MapGet(ctx, q.name, key)
			if err != nil {
				return nil, fmt.Errorf("map get %s: %w", q.name, err)
			}

			if !found {
				return nil, nil //nolint:nilnil // not-found is signalled as (nil result, nil error); see ExecuteTyped
			}

			return val, nil
		}

		return nil, unsupportedEngine(errUnsupportedMapReads, q.engine.Profile().Name)

	case ReadMembership:
		key := extractKeyValueByType(input, q.keyType)
		if sb, ok := q.engine.(SetBackend); ok {
			contained, err := sb.SetContains(ctx, q.name, key)
			if err != nil {
				return false, fmt.Errorf("set contains %s: %w", q.name, err)
			}

			return contained, nil
		}

		return nil, unsupportedEngine(errUnsupportedSetReads, q.engine.Profile().Name)

	case ReadFilteredScan:
		return s.executeFilteredScan(ctx, q, input)

	case ReadAggregate:
		if cb, ok := q.engine.(CounterBackend); ok {
			counts, err := cb.CounterGet(ctx, q.name)
			if err != nil {
				return nil, fmt.Errorf("counter get %s: %w", q.name, err)
			}

			return counts, nil
		}

		return nil, unsupportedEngine(errUnsupportedCounterReads, q.engine.Profile().Name)

	case ReadTraversal:
		node := extractKeyValueByType(input, q.keyType)
		if node == nil {
			node = extractFirstDomainField(input)
		}

		depth := extractDepthFromInput(input)
		if gb, ok := q.engine.(GraphBackend); ok {
			neighbors, err := gb.GraphNeighbors(ctx, q.name, node, depth)
			if err != nil {
				return nil, fmt.Errorf("graph neighbors %s: %w", q.name, err)
			}

			return neighbors, nil
		}

		return nil, unsupportedEngine(errUnsupportedGraphReads, q.engine.Profile().Name)

	case ReadMultiLookup:
		key := extractFirstDomainField(input)
		if mb, ok := q.engine.(MultimapBackend); ok {
			values, err := mb.MultiGet(ctx, q.name, key)
			if err != nil {
				return nil, fmt.Errorf("multi get %s: %w", q.name, err)
			}

			return values, nil
		}

		return nil, unsupportedEngine(errUnsupportedMultiReads, q.engine.Profile().Name)

	case ReadLogTail:
		limit := extractLimitFromInput(input)
		if lb, ok := q.engine.(LogBackend); ok {
			entries, err := lb.LogTail(ctx, q.name, limit)
			if err != nil {
				return nil, fmt.Errorf("log tail %s: %w", q.name, err)
			}

			return entries, nil
		}

		return nil, unsupportedEngine(errUnsupportedLogReads, q.engine.Profile().Name)

	case ReadScan:
		return s.executeFilteredScan(ctx, q, input)

	default:
		return nil, fmt.Errorf("%w: %s", errUnsupportedPattern, q.readPattern)
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
	filterPredicates := buildFilterPredicates(q, input)

	// Determine sort accessor.
	var sortFunc func(a, b any) int
	if q.config.sortAccessor.closure != nil {
		sortFunc = buildSortFunc(q.config.sortAccessor.closure)
	} else {
		// No explicit SortOn — try default: sort by first time.Time field.
		sortFunc = nil // engine decides internally
	}

	if sb, ok := q.engine.(ScanBackend); ok {
		rows, err := sb.MapScan(ctx, q.name, filterPredicates, sortFunc, cursorVal, limit)
		if err != nil {
			return nil, fmt.Errorf("map scan %s: %w", q.name, err)
		}

		return rows, nil
	}

	return nil, unsupportedEngine(errUnsupportedScanReads, q.engine.Profile().Name)
}

// buildFilterPredicates creates runtime filter predicates from typed closures
// declared via FilterOn. Each closure extracts a comparable value from a result
// item. The matching filter value is extracted from the query input by type.
func buildFilterPredicates(q queryRuntime, input any) []filterPredicate {
	if len(q.config.filterAccessors) == 0 {
		return nil
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

	return predicates
}

// extractValueByType finds a field in the input struct whose type matches
// the given type, and returns its value. Returns nil if not found or ambiguous.
func extractValueByType(input any, targetType reflect.Type) any {
	v, ok := structValue(input)
	if !ok {
		return nil
	}

	t := v.Type()

	metaNames := map[string]bool{limitField: true, afterField: true, depthField: true}

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
	ctx context.Context,
	store *Store,
	input Q,
) (R, error) {
	var zero R

	raw, err := store.ExecuteCtx(ctx, input)
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
		return zero, fmt.Errorf("%w: got %T", errExecuteTypeMismatch, raw)
	}

	return result, nil
}
