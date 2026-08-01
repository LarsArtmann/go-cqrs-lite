package metaengine

import (
	"context"
	"fmt"
	"reflect"
	"time"
)

// Execute dispatches a query input to its query's engine and returns the result.
func (s *Store) Execute(input any) (any, error) {
	return s.ExecuteCtx(context.Background(), input)
}

func (s *Store) ExecuteCtx(ctx context.Context, input any) (any, error) {
	s.readCount.Add(1)

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
	if err := s.IsPoisoned(q.name); err != nil {
		return nil, err
	}

	start := time.Now()
	result, err := s.executeQueryInner(ctx, q, input)

	if s.hooks != nil && s.hooks.OnExecute != nil {
		elapsed := time.Since(start)
		if s.hooks.SlowQueryThreshold == 0 || elapsed >= s.hooks.SlowQueryThreshold {
			s.hooks.OnExecute(q.name, q.readPattern, elapsed, err)
		}
	}

	return result, err
}

func (s *Store) executeQueryInner(
	ctx context.Context,
	q queryRuntime,
	input any,
) (any, error) {
	switch q.readPattern {
	case ReadPointLookup:
		key := extractKeyValueByType(input, q.keyType)

		// Fast path: raw JSON bytes → direct decode to R (1 JSON op instead of 3).
		if rvr, ok := q.engine.(RawValueReader); ok {
			raw, found, err := rvr.GetRawValue(ctx, q.name, key)
			if err != nil {
				return nil, fmt.Errorf("raw get %s: %w", q.name, err)
			}

			if !found {
				return nil, nil //nolint:nilnil // not-found signalled as (nil, nil)
			}

			return jsonValue(raw), nil
		}

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

	// Fastest path: raw JSON bytes per row → direct decode to element type
	// (1 JSON op per row instead of 3). Preferred when the engine supports it
	// and all filter/sort accessors are declarative (pushdown-eligible).
	if rsr, ok := q.engine.(RawScanReader); ok && canPushdown(q.config) {
		specs := buildFilterSpecs(q.config, input)

		var sortSpec *SortSpec
		if q.config.sortAccessor.spec != nil {
			sortSpec = q.config.sortAccessor.spec
		}

		rawRows, err := rsr.ScanRawValues(ctx, q.name, specs, sortSpec, cursorVal, limit)
		if err != nil {
			return nil, fmt.Errorf("raw scan %s: %w", q.name, err)
		}

		result := make([]any, len(rawRows))
		for i, raw := range rawRows {
			result[i] = jsonValue(raw)
		}

		return result, nil
	}

	// Fast path: if the engine supports pushdown AND all filter/sort accessors
	// have declarative specs (FilterOnField/SortOnField), push WHERE/ORDER BY/
	// LIMIT into SQL instead of loading all rows into Go.
	if pushdown, ok := q.engine.(PushdownScan); ok && canPushdown(q.config) {
		specs := buildFilterSpecs(q.config, input)

		var sortSpec *SortSpec
		if q.config.sortAccessor.spec != nil {
			sortSpec = q.config.sortAccessor.spec
		}

		rows, err := pushdown.PushdownMapScan(ctx, q.name, specs, sortSpec, cursorVal, limit)
		if err != nil {
			return nil, fmt.Errorf("pushdown map scan %s: %w", q.name, err)
		}

		return rows, nil
	}

	// Fallback: closure-based in-Go filtering via ScanBackend.MapScan.
	filterPredicates := buildFilterPredicates(q, input)

	var filterFn func(item any) bool
	if len(filterPredicates) > 0 {
		filterFn = func(item any) bool {
			return passesFilters(item, filterPredicates)
		}
	}

	var sortFunc func(a, b any) int
	if q.config.sortAccessor.closure != nil {
		sortFunc = buildSortFunc(q.config.sortAccessor.closure)
	} else {
		sortFunc = nil
	}

	if sb, ok := q.engine.(ScanBackend); ok {
		rows, err := sb.MapScan(ctx, q.name, filterFn, sortFunc, cursorVal, limit)
		if err != nil {
			return nil, fmt.Errorf("map scan %s: %w", q.name, err)
		}

		return rows, nil
	}

	return nil, unsupportedEngine(errUnsupportedScanReads, q.engine.Profile().Name)
}

// canPushdown returns true when all declared filter/sort accessors carry
// declarative specs (FilterSpec/SortSpec). If any accessor is closure-only
// (FilterOn/SortOn), pushdown is impossible and the fallback path is used.
func canPushdown(cfg QueryConfig) bool {
	for _, acc := range cfg.filterAccessors {
		if acc.spec == nil {
			return false
		}
	}

	if cfg.sortAccessor.closure != nil && cfg.sortAccessor.spec == nil {
		return false
	}

	return true
}

// buildFilterSpecs converts declarative filter accessors into FilterSpec values
// with the filter values extracted from the query input by field name.
func buildFilterSpecs(cfg QueryConfig, input any) []FilterSpec {
	if len(cfg.filterAccessors) == 0 {
		return nil
	}

	var specs []FilterSpec

	for _, acc := range cfg.filterAccessors {
		if acc.spec == nil {
			continue
		}

		val := extractValueByName(input, acc.spec.Column)
		if val == nil {
			continue
		}

		specs = append(specs, FilterSpec{
			Column: acc.spec.Column,
			Op:     acc.spec.Op,
			Value:  val,
		})
	}

	return specs
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
		// Declarative filters (FilterOnField) carry a spec but no closure. They
		// must still be honored in the closure-fallback path, otherwise mixing a
		// declarative filter with a closure sort silently drops the filter. The
		// expected value is read from the input by column name; the item value is
		// read from each row by the same column name (map key or struct field).
		if acc.spec != nil {
			expected := extractValueByName(input, acc.spec.Column)
			if expected == nil {
				continue
			}

			col := acc.spec.Column

			predicates = append(predicates, filterPredicate{
				expected: expected,
				test: func(item any) bool {
					return reflect.DeepEqual(itemFieldByName(item, col), expected)
				},
			})

			continue
		}

		// Extract the expected filter value from the input by type matching.
		expected := extractValueByType(input, acc.returnType)
		if expected == nil {
			// No matching field in input — skip this filter (no constraint).
			continue
		}

		closure := acc.closure
		closureParam := reflect.TypeOf(closure).In(0)

		predicates = append(predicates, filterPredicate{
			expected: expected,
			test: func(item any) bool {
				closureVal := reflect.ValueOf(closure)
				// SQL engines decode struct rows as map[string]any; reify to the
				// typed parameter or reflect.Call panics on the type mismatch.
				result := closureVal.Call([]reflect.Value{reifyReflect(item, closureParam)})

				return reflect.DeepEqual(result[0].Interface(), expected)
			},
		})
	}

	return predicates
}

// extractValueByType finds a field in the input struct whose type matches
// the given type, and returns its value. Returns nil if not found or ambiguous.
// extractValueByType finds a non-meta field in the input struct whose type
// matches targetType, and returns its value. Returns nil if not found or
// ambiguous.
func extractValueByType(input any, targetType reflect.Type) any {
	metaNames := map[string]bool{limitField: true, afterField: true, depthField: true}

	return findValueByType(input, targetType, func(name string) bool { return metaNames[name] })
}

// buildSortFunc creates a comparator from a SortOn closure.
// The closure extracts a sort key from each result item.
// When comparing against a cursor value (which is already a raw sort key,
// not a result item), the closure is skipped and the value is used directly.
func buildSortFunc(closure any) func(a, b any) int {
	closureVal := reflect.ValueOf(closure)
	paramType := closureVal.Type().In(0)

	extractKey := func(item any) any {
		t := reflect.TypeOf(item)
		if t == paramType {
			return closureVal.Call([]reflect.Value{reflect.ValueOf(item)})[0].Interface()
		}
		// SQL engines decode struct rows as map[string]any; reify to paramType
		// so the SortOn closure extracts the real sort key.
		if t != nil && t.Kind() == reflect.Map {
			return closureVal.Call([]reflect.Value{reifyReflect(item, paramType)})[0].Interface()
		}

		return item // raw cursor key — compared directly
	}

	return func(a, b any) int {
		return compareValue(extractKey(a), extractKey(b))
	}
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

	closureVal := reflect.ValueOf(q.config.sortAccessor.closure)
	paramType := closureVal.Type().In(0)

	return func(item any) any {
		return closureVal.Call([]reflect.Value{reifyReflect(item, paramType)})[0].Interface()
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
		return zero, ErrNotFound
	}

	if isCollectionResult[R]() {
		limit := extractLimitFromInput(input)
		sortFn := store.sortKeyFn(qualifiedTypeName(input))

		return reconstructCollection[R](raw, limit, sortFn), nil
	}

	result, ok := raw.(R)
	if !ok {
		// SQLite engines return map[string]any for struct values (JSON round-trip
		// through any). Re-reify via JSON so struct-typed results work across
		// both memory and SQL engines. Memory values pass the direct assertion
		// above; the reify cost is paid only on the SQLite path.
		if reified, rErr := reify[R](raw); rErr == nil {
			return reified, nil
		}

		return zero, fmt.Errorf("%w: got %T", errExecuteTypeMismatch, raw)
	}

	return result, nil
}
