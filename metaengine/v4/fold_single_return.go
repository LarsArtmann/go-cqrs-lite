package metaengine

import (
	"fmt"
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// reflectCall1 creates a pre-bound closure that calls a single-param,
// single-return reflect.Value and type-asserts the result to T.
// The reflect.Value is captured once at construction time; the hot path
// does not call reflect.ValueOf(handler) per event. Plain On handlers have
// no Record parameter, so the closure discards it.
func reflectCall1[T any](hv reflect.Value) func(record.Record, any) T {
	return func(_ record.Record, event any) T {
		return hv.Call([]reflect.Value{reflect.ValueOf(event)})[0].Interface().(T)
	}
}

func classifySingleReturn[E any](
	sample E,
	eventType string,
	outType reflect.Type,
	hv reflect.Value,
) Fold {
	switch outType {
	case reflect.TypeFor[Embedding]():
		return &vectorFold{
			eventType: eventType,
			sample:    sample,
			invoke:    reflectCall1[Embedding](hv),
		}

	case reflect.TypeFor[IndexedText]():
		return &searchFold{
			eventType: eventType,
			sample:    sample,
			invoke:    reflectCall1[IndexedText](hv),
		}

	case reflect.TypeFor[Point]():
		return &spatialFold{eventType: eventType, sample: sample, invoke: reflectCall1[Point](hv)}

	case reflect.TypeFor[Delta]():
		return &countFold{eventType: eventType, sample: sample, invoke: reflectCall1[Delta](hv)}

	case reflect.TypeFor[Edge]():
		return &edgeFold{eventType: eventType, sample: sample, invoke: reflectCall1[Edge](hv)}

	case reflect.TypeFor[EdgeRemoval]():
		return &edgeRemoveFold{
			eventType: eventType,
			sample:    sample,
			invoke:    reflectCall1[EdgeRemoval](hv),
		}

	case reflect.TypeFor[Skip]():
		return &skipFold{eventType: eventType, sample: sample}

	case reflect.TypeFor[MultiEntry]():
		return &multiInsertFold{
			eventType: eventType,
			sample:    sample,
			invoke:    reflectCall1[MultiEntry](hv),
		}

	case reflect.TypeFor[Append]():
		return &appendFold{eventType: eventType, sample: sample, invoke: reflectCall1[Append](hv)}

	default:
		rejectSliceOfFoldUnit("On", eventType, outType)

		return &setFold{
			eventType: eventType,
			sample:    sample,
			keyType:   outType,
			invoke: func(_ record.Record, event any) any {
				return hv.Call([]reflect.Value{reflect.ValueOf(event)})[0].Interface()
			},
		}
	}
}

// canonicalFoldUnits are the single-value fold results the constructors
// switch on: a handler returning ONE of these per event is classified into
// its dedicated fold kind. Used by rejectSliceOfFoldUnit to spot the batch
// lookalikes ([]Edge, []Embedding, ...) that must never reach the Set
// fallback.
var canonicalFoldUnits = []reflect.Type{
	reflect.TypeFor[Delta](),
	reflect.TypeFor[Edge](),
	reflect.TypeFor[EdgeRemoval](),
	reflect.TypeFor[Embedding](),
	reflect.TypeFor[IndexedText](),
	reflect.TypeFor[Point](),
	reflect.TypeFor[Skip](),
	reflect.TypeFor[MultiEntry](),
	reflect.TypeFor[Append](),
}

// rejectSliceOfFoldUnit panics when a single-return fold handler returns a
// slice whose element type is a canonical fold unit ([]Edge, []Embedding,
// ...). The shape reads as a batch fan-out, but the fold contract produces
// ONE unit per event: routing it through the Set fallback would silently
// misfile the whole slice as a set key and drop every unit while
// construction, planning, apply, and rebuild all report success (issue
// #130). Custom key types keep flowing through the Set fallback unchanged.
func rejectSliceOfFoldUnit(construct, eventType string, outType reflect.Type) {
	if outType.Kind() != reflect.Slice {
		return
	}

	for _, unit := range canonicalFoldUnits {
		if outType.Elem() != unit {
			continue
		}

		panic(fmt.Sprintf(
			"metaengine.%s(%s): handler return type %s is a slice of a canonical fold unit — a fold produces ONE unit per event, so this shape would be silently misrouted into a Set collection keyed by the whole slice (issue #130). Emit one event per relation at the write side, or fold the batch into a Map/Multimap collection and derive the units in a second pass",
			construct,
			eventType,
			outType,
		))
	}
}
