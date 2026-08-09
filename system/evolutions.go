package system

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// EvolutionSpec is a sealed interface for evolution declarations. An Evolution
// declares how a result type materializes from events — the materialization
// rule (fold). Only [Evolve] produces EvolutionSpec values.
type EvolutionSpec interface {
	isEvolutionSpec()
}

// evolutionSpec carries folds and decoder entries for a result type.
type evolutionSpec struct {
	name          string
	resultType    reflect.Type
	keyField      string
	samples       []metaengine.NamedSample
	explicitFolds []explicitFoldEntry
	internal      bool
}

func (*evolutionSpec) isEvolutionSpec() {}

// explicitFoldEntry stores a type-erased fold for non-convention events.
type explicitFoldEntry struct {
	eventType string
	sample    any
	mutate    func(event any, result any) // func(E, *R), type-erased
}

// EvolveOption tunes an Evolution declaration.
type EvolveOption func(*evolveOptConfig)

type evolveOptConfig struct {
	keyField string
	internal bool
}

// Internal marks the Evolution as state-only — not queryable via Get or Find,
// but available for command state loading.
func Internal() EvolveOption {
	return func(c *evolveOptConfig) { c.internal = true }
}

// EvolveKey overrides the key field name (default: "ID").
func EvolveKey(field string) EvolveOption {
	return func(c *evolveOptConfig) { c.keyField = field }
}

// evolutionBuilder declares how result type R emerges from events.
type evolutionBuilder[R any] struct {
	name          string
	keyField      string
	samples       []metaengine.NamedSample
	explicitFolds []explicitFoldEntry
	internal      bool
}

// Evolve declares how result type R materializes from events.
//
// Level 1 — convention (Created/Updated/Deleted suffix → auto fold):
//
//	system.Evolve[TaskView]("tasks").
//	    On("task.created", TaskCreated{}).
//	    On("task.deleted", TaskDeleted{}).
//	    Done()
//
// Level 2 — explicit fold (for non-convention events):
//
//	system.Evolve[TaskView]("tasks").
//	    On("task.completed", TaskCompleted{},
//	        func(e TaskCompleted, v *TaskView) { v.Status = "done" }).
//	    Done()
func Evolve[R any](name string, opts ...EvolveOption) *evolutionBuilder[R] {
	cfg := evolveOptConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &evolutionBuilder[R]{
		name:     name,
		keyField: cfg.keyField,
		internal: cfg.internal,
	}
}

// OnEvolution registers an event handler for the builder. E is inferred from the sample.
// Zero fold functions → convention fold. One fold function → explicit mutation.
// This is a standalone function (not a method) because Go does not allow type
// parameters on methods.
func OnEvolution[R any, E any](
	b *evolutionBuilder[R],
	eventType string,
	sample E,
	fold ...func(E, *R),
) *evolutionBuilder[R] {
	if len(fold) == 0 {
		b.samples = append(b.samples, metaengine.NamedEvent(eventType, sample))

		return b
	}

	fn := fold[0]

	b.explicitFolds = append(b.explicitFolds, explicitFoldEntry{
		eventType: eventType,
		sample:    sample,
		mutate: func(event any, result any) {
			fn(event.(E), result.(*R))
		},
	})

	return b
}

// Done finalizes the Evolution declaration.
func (b *evolutionBuilder[R]) Done() EvolutionSpec {
	keyField := b.keyField
	if keyField == "" {
		keyField = "ID"
	}

	return &evolutionSpec{
		name:          b.name,
		resultType:    reflect.TypeFor[R](),
		keyField:      keyField,
		samples:       append([]metaengine.NamedSample(nil), b.samples...),
		explicitFolds: append([]explicitFoldEntry(nil), b.explicitFolds...),
		internal:      b.internal,
	}
}

// makeExplicitFold creates a metaengine update fold from an explicit fold entry.
// It reifies prev to the result type, applies the mutation, and returns the result.
func makeExplicitFold(resultType reflect.Type, ef explicitFoldEntry) metaengine.Fold {
	return metaengine.OnTyped(ef.eventType, ef.sample, func(e any, prev any) any {
		v := reflect.New(resultType)

		if prev != nil {
			reifyTo(prev, v.Interface())
		}

		ef.mutate(e, v.Interface())

		return v.Elem().Interface()
	})
}

// reifyTo converts a raw value (potentially map[string]any from JSON engines)
// into the target. For Memory engine, the direct type assignment succeeds.
func reifyTo(src, dst any) {
	if src == nil {
		return
	}

	srcVal := reflect.ValueOf(src)
	dstVal := reflect.ValueOf(dst)

	if srcVal.Type() == dstVal.Elem().Type() {
		dstVal.Elem().Set(srcVal)

		return
	}

	data, err := json.Marshal(src)
	if err != nil {
		return
	}

	_ = json.Unmarshal(data, dst) //nolint:errcheck // best-effort reify
}

// evolutionDecoderEntries extracts decoder entries from an evolutionSpec's
// samples and explicit folds.
func evolutionDecoderEntries(evo *evolutionSpec) []decoderEntry {
	var entries []decoderEntry

	for _, s := range evo.samples {
		entries = append(entries, decoderEntry{eventType: s.EventType(), sample: s.Sample()})
	}

	for _, ef := range evo.explicitFolds {
		entries = append(entries, decoderEntry{eventType: ef.eventType, sample: ef.sample})
	}

	return entries
}

// buildEvolutionFolds generates metaengine.Fold values from an evolution spec.
// Convention events (Created/Updated/Deleted suffix) produce auto-CRUD folds;
// explicit folds use makeExplicitFold.
func buildEvolutionFolds[R any](evo *evolutionSpec) ([]metaengine.Fold, error) {
	var folds []metaengine.Fold

	if len(evo.samples) > 0 {
		auto, err := metaengine.AutoCRUDByNamedEvents[R](evo.keyField, evo.samples...)
		if err != nil {
			return nil, fmt.Errorf("system: evolution %q: %w", evo.name, err)
		}

		folds = append(folds, auto...)
	}

	for _, ef := range evo.explicitFolds {
		folds = append(folds, makeExplicitFold(evo.resultType, ef))
	}

	return folds, nil
}

// buildQueryFromFolds creates a metaengine.Query from fold functions and decoder entries.
func buildQueryFromFolds[Q any, R any](
	name string,
	folds []metaengine.Fold,
	entries []decoderEntry,
) (any, []decoderEntry, error) {
	foldArgs := make([]any, len(folds))
	for i, f := range folds {
		foldArgs[i] = f
	}

	query := metaengine.Query[Q, R](name, foldArgs...)

	return query, entries, nil
}
