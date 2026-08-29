package metaengine

import (
	"fmt"
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// RecordAwareFold is a historical extension point: it allowed folds to receive
// the full Record (Type, Payload, StreamID, StreamType, Version, MetaData)
// via SetCurrentRecord before their invoke closure ran (ADR-0112).
//
// Since 2026-08-29 the Record is passed to fold invocations as a VALUE through
// the invoke closure instead of a shared mutable cell, so the engine's own
// folds no longer implement this interface — a fold-internal cell was shared
// mutable state across Stores planned from the same declarations (data race
// plus cross-attribution). Kept for source compatibility; removal in v5.
//
// Deprecated: OnRecord handlers receive record.Record as their first
// parameter directly; do not implement this interface.
type RecordAwareFold interface {
	SetCurrentRecord(r record.Record)
}

// OnRecord creates a Record-aware fold. The handler receives both the decoded
// payload AND the full Record context, enabling folds that need StreamID,
// Version, or metadata for projection logic.
//
// Handler signatures (same shapes as On, but with an additional record.Record
// first parameter):
//
//	metaengine.OnRecord(Event{}, func(rec record.Record, e Event) (Key, Value) { ... })
//	metaengine.OnRecord(Event{}, func(rec record.Record, e Event, prev Value) Value { ... })
//	metaengine.OnRecord(Event{}, func(rec record.Record, e Event) metaengine.Delta { ... })
//	metaengine.OnRecord(Event{}, func(rec record.Record, e Event) metaengine.Edge { ... })
//
// The Record parameter is always first, followed by the typed payload.
func OnRecord[E any](sample E, handler any) Fold {
	return onRecordFold(EventTypeName(sample), sample, handler)
}

// OnRecordTyped is like OnRecord but binds to an explicit event-type string.
func OnRecordTyped[E any](eventType string, sample E, handler any) Fold {
	return onRecordFold(eventType, sample, handler)
}

func onRecordFold[E any](eventType string, sample E, handler any) Fold {
	if rs, ok := handler.(removeSignal); ok {
		return &removeFold{eventType: eventType, sample: sample, valueType: rs.valueType}
	}

	handlerType := reflect.TypeOf(handler)
	if handlerType == nil || handlerType.Kind() != reflect.Func {
		panic(fmt.Sprintf(
			"metaengine.OnRecord(%s): handler must be a function, got %T",
			eventType, handler,
		))
	}

	if handlerType.NumIn() < 2 {
		panic(fmt.Sprintf(
			"metaengine.OnRecord(%s): handler must have at least 2 params (record.Record, E), got %d",
			eventType,
			handlerType.NumIn(),
		))
	}

	recordType := handlerType.In(0)
	if recordType != reflect.TypeFor[record.Record]() {
		panic(fmt.Sprintf(
			"metaengine.OnRecord(%s): first param must be record.Record, got %v",
			eventType, recordType,
		))
	}

	if err := verifyRecordEventParam[E](handlerType, eventType); err != nil {
		panic(err.Error())
	}

	hv := reflect.ValueOf(handler)
	numIn := handlerType.NumIn()
	numOut := handlerType.NumOut()

	callWithRecord := func(rec record.Record, payload any) []reflect.Value {
		return hv.Call([]reflect.Value{
			reflect.ValueOf(rec),
			reflect.ValueOf(payload),
		})
	}

	switch {
	case numIn == 2 && numOut == 2:
		invoke := func(rec record.Record, event any) (any, any) {
			results := callWithRecord(rec, event)
			return results[0].Interface(), results[1].Interface()
		}
		f := &insertFold{
			eventType: eventType,
			sample:    sample,
			keyType:   handlerType.Out(0),
			valueType: handlerType.Out(1),
			invoke:    invoke,
		}
		return f

	case numIn == 3 && numOut == 1:
		invoke := func(rec record.Record, event, prev any) any {
			args := []reflect.Value{
				reflect.ValueOf(rec),
				reflect.ValueOf(event),
			}
			prevType := hv.Type().In(2)
			if prev != nil {
				args = append(args, reifyReflect(prev, prevType))
			} else {
				args = append(args, reflect.Zero(prevType))
			}
			return hv.Call(args)[0].Interface()
		}
		f := &updateFold{
			eventType: eventType,
			sample:    sample,
			valueType: handlerType.Out(0),
			invoke:    invoke,
		}
		return f

	case numIn == 2 && numOut == 1:
		outType := handlerType.Out(0)

		switch outType {
		case reflect.TypeFor[Delta]():
			invoke := func(rec record.Record, event any) Delta {
				return callWithRecord(rec, event)[0].Interface().(Delta)
			}
			return &countFold{eventType: eventType, sample: sample, invoke: invoke}

		case reflect.TypeFor[Edge]():
			invoke := func(rec record.Record, event any) Edge {
				return callWithRecord(rec, event)[0].Interface().(Edge)
			}
			f := &edgeFold{eventType: eventType, sample: sample, invoke: invoke}
			return f

		case reflect.TypeFor[EdgeRemoval]():
			invoke := func(rec record.Record, event any) EdgeRemoval {
				return callWithRecord(rec, event)[0].Interface().(EdgeRemoval)
			}
			f := &edgeRemoveFold{eventType: eventType, sample: sample, invoke: invoke}
			return f

		case reflect.TypeFor[Embedding]():
			invoke := func(rec record.Record, event any) Embedding {
				return callWithRecord(rec, event)[0].Interface().(Embedding)
			}
			return &vectorFold{eventType: eventType, sample: sample, invoke: invoke}

		case reflect.TypeFor[IndexedText]():
			invoke := func(rec record.Record, event any) IndexedText {
				return callWithRecord(rec, event)[0].Interface().(IndexedText)
			}
			return &searchFold{eventType: eventType, sample: sample, invoke: invoke}

		case reflect.TypeFor[Point]():
			invoke := func(rec record.Record, event any) Point {
				return callWithRecord(rec, event)[0].Interface().(Point)
			}
			return &spatialFold{eventType: eventType, sample: sample, invoke: invoke}

		case reflect.TypeFor[Skip]():
			return &skipFold{eventType: eventType, sample: sample}

		case reflect.TypeFor[MultiEntry]():
			invoke := func(rec record.Record, event any) MultiEntry {
				return callWithRecord(rec, event)[0].Interface().(MultiEntry)
			}
			return &multiInsertFold{eventType: eventType, sample: sample, invoke: invoke}

		case reflect.TypeFor[Append]():
			invoke := func(rec record.Record, event any) Append {
				return callWithRecord(rec, event)[0].Interface().(Append)
			}
			return &appendFold{eventType: eventType, sample: sample, invoke: invoke}

		default:
			invoke := func(rec record.Record, event any) any {
				return callWithRecord(rec, event)[0].Interface()
			}
			f := &setFold{eventType: eventType, sample: sample, keyType: outType, invoke: invoke}
			return f
		}
	}

	panic(fmt.Sprintf(
		"metaengine.OnRecord(%s): unsupported handler signature %v (see OnRecord docs)",
		eventType, handlerType,
	))
}
