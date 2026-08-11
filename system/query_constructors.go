package system

import (
	"fmt"
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// ─── Lookup: point lookup by key ───

// lookupBuilder builds a point-lookup projection. The engine builds a hash
// map keyed by ID, giving O(1) reads.
type lookupBuilder[R any] struct {
	name           string
	keyField       string
	samples        []metaengine.NamedSample
	layoutPriority metaengine.Priority
}

// Lookup declares a point-lookup projection: read a single row by key.
// The engine builds a hash map for O(1) point reads.
//
//	system.Lookup[UserView]("get-user").
//	    On("user.created", UserCreated{}).
//	    On("user.updated", UserUpdated{}).
//	    On("user.deleted", UserDeleted{}).
//	    Done()
func Lookup[R any](name string) *lookupBuilder[R] {
	return &lookupBuilder[R]{name: name}
}

// Key overrides the key field name (default: "ID").
func (b *lookupBuilder[R]) Key(field string) *lookupBuilder[R] {
	b.keyField = field

	return b
}

// On registers an event type and sample struct for auto fold generation.
// The sample struct name suffix (Created/Updated/Deleted) classifies the fold
// kind. Chainable. Finalize with [.Done].
func (b *lookupBuilder[R]) On(eventType string, sample any) *lookupBuilder[R] {
	b.samples = append(b.samples, metaengine.NamedEvent(eventType, sample))

	return b
}

// Priority pins the layout-planning objective for this query
// (ADR-0124 Layer 4). The operator's per-query PriorityConfig still takes
// precedence; this is the developer-side per-query override for queries
// with a different optimization objective than the deployment default.
func (b *lookupBuilder[R]) Priority(p metaengine.Priority) *lookupBuilder[R] {
	b.layoutPriority = p

	return b
}

// Done finalizes the projection declaration.
func (b *lookupBuilder[R]) Done() ProjectionDeclaration {
	keyField := b.keyField
	if keyField == "" {
		keyField = "ID"
	}

	name := b.name
	rt := reflect.TypeFor[R]()
	samplesCopy := append([]metaengine.NamedSample(nil), b.samples...)

	return ProjectionSpec{
		name:       name,
		resultType: rt,
		build: func(evoIndex map[reflect.Type]*evolutionSpec) (any, []decoderEntry, error) {
			if len(samplesCopy) > 0 {
				return buildCRUDQuery[LookupInput[string], R](name, keyField, samplesCopy, layoutPriority)
			}

			if evo, ok := evoIndex[rt]; ok {
				folds, err := buildEvolutionFolds[R](evo)
				if err != nil {
					return nil, nil, err
				}

				q, decs := buildQueryFromFolds[LookupInput[string], R](
					name,
					folds,
					evolutionDecoderEntries(evo),
				)
				return q, decs, nil
			}

			return nil, nil, fmt.Errorf(
				"system: projection %q: no samples and no matching evolution",
				name,
			)
		},
	}
}

// ─── QuerySet: filtered, sorted, paginated collection ───

// querySetBuilder builds a collection projection with flexible runtime
// filtering and sorting. The engine builds a table with indexes on declared
// filterable/sortable fields.
type querySetBuilder[R any] struct {
	name       string
	keyField   string
	samples    []metaengine.NamedSample
	filterable []string
	sortFields []sortSpec
}

type sortSpec struct {
	field string
	desc  bool
}

// QuerySet declares a collection projection with flexible runtime filters.
// Declare filterable and sortable fields at build time; the engine generates
// indexes. At runtime, use any combination of declared filters.
//
//	system.QuerySet[TaskView]("tasks").
//	    On("task.created", TaskCreated{}).
//	    On("task.deleted", TaskDeleted{}).
//	    Filterable("status", "priority").
//	    Sortable("priority", true).
//	    Done()
func QuerySet[R any](name string) *querySetBuilder[R] {
	return &querySetBuilder[R]{name: name}
}

// Key overrides the key field name (default: "ID").
func (b *querySetBuilder[R]) Key(field string) *querySetBuilder[R] {
	b.keyField = field

	return b
}

// On registers an event type and sample struct for auto fold generation.
func (b *querySetBuilder[R]) On(eventType string, sample any) *querySetBuilder[R] {
	b.samples = append(b.samples, metaengine.NamedEvent(eventType, sample))

	return b
}

// Filterable declares fields that runtime queries can filter on.
// The engine generates indexes on these fields for SQL pushdown.
func (b *querySetBuilder[R]) Filterable(fields ...string) *querySetBuilder[R] {
	b.filterable = append(b.filterable, fields...)

	return b
}

// Sortable declares a field that runtime queries can sort by.
// desc=true means descending order by default.
func (b *querySetBuilder[R]) Sortable(field string, desc bool) *querySetBuilder[R] {
	b.sortFields = append(b.sortFields, sortSpec{field: field, desc: desc})

	return b
}

// Done finalizes the projection declaration.
func (b *querySetBuilder[R]) Done() ProjectionDeclaration {
	keyField := b.keyField
	if keyField == "" {
		keyField = "ID"
	}

	name := b.name
	rt := reflect.TypeFor[R]()
	samplesCopy := append([]metaengine.NamedSample(nil), b.samples...)
	filterCopy := append([]string(nil), b.filterable...)
	sortCopy := append([]sortSpec(nil), b.sortFields...)

	return ProjectionSpec{
		name:       name,
		resultType: rt,
		build: func(evoIndex map[reflect.Type]*evolutionSpec) (any, []decoderEntry, error) {
			opts := make([]any, 0, len(filterCopy)+len(sortCopy))

			for _, f := range filterCopy {
				opts = append(opts, metaengine.FilterOnField[R](f, metaengine.FilterEq))
			}

			for _, s := range sortCopy {
				opts = append(opts, metaengine.SortOnField[R](s.field, s.desc))
			}

			if len(samplesCopy) > 0 {
				return buildCRUDQueryWithOptions[ScanInput, R](name, keyField, samplesCopy, opts)
			}

			if evo, ok := evoIndex[rt]; ok {
				folds, err := buildEvolutionFolds[R](evo)
				if err != nil {
					return nil, nil, err
				}

				args := make([]any, 0, len(folds)+len(opts))
				for _, f := range folds {
					args = append(args, f)
				}
				args = append(args, opts...)

				query := metaengine.Query[ScanInput, R](name, args...)

				return query, evolutionDecoderEntries(evo), nil
			}

			return nil, nil, fmt.Errorf(
				"system: projection %q: no samples and no matching evolution",
				name,
			)
		},
	}
}

// ─── Count: counter aggregate ───

// countEntry pairs a wire event type with a counter delta.
type countEntry struct {
	eventType string
	sample    any
	delta     int64
	key       string
}

// countBuilder builds a counter projection that maintains numeric aggregates.
type countBuilder struct {
	name    string
	entries []countEntry
}

// Count declares a counter projection: maintain numeric aggregates per key.
// Each .On call registers an event that increments or decrements a counter key.
//
//	system.Count("task-counts").
//	    On("task.created", TaskCreated{}, +1, "pending").
//	    On("task.completed", TaskCompleted{}, -1, "active").
//	    On("task.completed", TaskCompleted{}, +1, "done").
//	    Done()
func Count(name string) *countBuilder {
	return &countBuilder{name: name}
}

// On registers an event that adjusts a counter key by delta.
// The sample provides the event struct type for decoder construction.
func (b *countBuilder) On(eventType string, sample any, delta int64, key string) *countBuilder {
	b.entries = append(b.entries, countEntry{
		eventType: eventType,
		sample:    sample,
		delta:     delta,
		key:       key,
	})

	return b
}

// Done finalizes the projection declaration.
func (b *countBuilder) Done() ProjectionDeclaration {
	name := b.name
	entriesCopy := append([]countEntry(nil), b.entries...)

	return ProjectionSpec{
		name:       name,
		resultType: reflect.TypeFor[map[string]int64](),
		build: func(_ map[reflect.Type]*evolutionSpec) (any, []decoderEntry, error) {
			q, decs := buildCounterQuery(name, entriesCopy)
			return q, decs, nil
		},
	}
}

// ─── Shared build helpers ───

// buildCRUDQuery generates a CRUD query declaration from named event samples.
func buildCRUDQuery[Q any, R any](
	name, keyField string,
	samples []metaengine.NamedSample,
) (any, []decoderEntry, error) {
	folds, err := metaengine.AutoCRUDByNamedEvents[R](keyField, samples...)
	if err != nil {
		return nil, nil, fmt.Errorf("system: projection %q: %w", name, err)
	}

	foldArgs := make([]any, len(folds))
	for i, f := range folds {
		foldArgs[i] = f
	}

	query := metaengine.Query[Q, R](name, foldArgs...)

	return query, samplesToDecoderEntries(samples), nil
}

// buildCRUDQueryWithOptions is like buildCRUDQuery but appends QueryOption
// values (FilterOnField, SortOnField) after the folds.
func buildCRUDQueryWithOptions[Q any, R any](
	name, keyField string,
	samples []metaengine.NamedSample,
	opts []any,
) (any, []decoderEntry, error) {
	folds, err := metaengine.AutoCRUDByNamedEvents[R](keyField, samples...)
	if err != nil {
		return nil, nil, fmt.Errorf("system: projection %q: %w", name, err)
	}

	args := make([]any, 0, len(folds)+len(opts))
	for _, f := range folds {
		args = append(args, f)
	}

	args = append(args, opts...)

	query := metaengine.Query[Q, R](name, args...)

	return query, samplesToDecoderEntries(samples), nil
}

// buildCounterQuery generates a counter query declaration from count entries.
func buildCounterQuery(
	name string,
	entries []countEntry,
) (any, []decoderEntry) {
	folds := make([]metaengine.Fold, 0, len(entries))
	decEntries := make([]decoderEntry, 0, len(entries))

	for _, e := range entries {
		delta := e.delta
		key := e.key

		fold := metaengine.OnRecordTyped(
			e.eventType,
			e.sample,
			func(_ record.Record, _ any) metaengine.Delta {
				return metaengine.Delta{key: delta}
			},
		)
		folds = append(folds, fold)

		decEntries = append(decEntries, decoderEntry{
			eventType: e.eventType,
			sample:    e.sample,
		})
	}

	foldArgs := make([]any, len(folds))
	for i, f := range folds {
		foldArgs[i] = f
	}

	query := metaengine.Query[CountInput, map[string]int64](name, foldArgs...)

	return query, decEntries
}

func samplesToDecoderEntries(samples []metaengine.NamedSample) []decoderEntry {
	entries := make([]decoderEntry, len(samples))
	for i, s := range samples {
		entries[i] = decoderEntry{
			eventType: s.EventType(),
			sample:    s.Sample(),
		}
	}

	return entries
}

// Event is an alias for [metaengine.NamedEvent], exported for ergonomic
// one-package import when declaring projections.
func Event(eventType string, sample any) metaengine.NamedSample {
	return metaengine.NamedEvent(eventType, sample)
}
