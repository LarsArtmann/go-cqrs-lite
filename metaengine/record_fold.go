package metaengine

import (
	"fmt"
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// RecordAwareFold is an optional interface implemented by folds that need
// access to the full Record (Type, Payload, StreamID, StreamType, Version,
// MetaData) rather than just the decoded payload (ADR-0112).
//
// When store.ApplyRecord is called, Record-aware folds receive the Record via
// SetCurrentRecord before their invoke closure runs. When store.Apply is called
// (the legacy path), a minimal Record is synthesized — metadata fields are
// zero-valued.
type RecordAwareFold interface {
	SetCurrentRecord(record.Record)
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
			eventType, handlerType.NumIn(),
		))
	}

	recordType := handlerType.In(0)
	if recordType != reflect.TypeOf(record.Record{}) {
		panic(fmt.Sprintf(
			"metaengine.OnRecord(%s): first param must be record.Record, got %v",
			eventType, recordType,
		))
	}

	hv := reflect.ValueOf(handler)
	numIn := handlerType.NumIn()
	numOut := handlerType.NumOut()

	// currentRecord is captured by reference so closures see updates.
	recHolder := &struct{ rec record.Record }{}

	callWithRecord := func(payload any) []reflect.Value {
		return hv.Call([]reflect.Value{
			reflect.ValueOf(recHolder.rec),
			reflect.ValueOf(payload),
		})
	}

	switch {
	case numIn == 2 && numOut == 2:
		invoke := func(event any) (any, any) {
			results := callWithRecord(event)
			return results[0].Interface(), results[1].Interface()
		}
		f := &insertFold{
			eventType: eventType,
			sample:    sample,
			keyType:   handlerType.Out(0),
			valueType: handlerType.Out(1),
			invoke:    invoke,
		}
		f.recordSetter = func(r record.Record) { recHolder.rec = r }
		return f

	case numIn == 3 && numOut == 1:
		invoke := func(event, prev any) any {
			args := []reflect.Value{
				reflect.ValueOf(recHolder.rec),
				reflect.ValueOf(event),
			}
			prevType := hv.Type().In(2)
			if prev != nil {
				args = append(args, reflect.ValueOf(prev))
			} else {
				args = append(args, reflect.Zero(prevType))
			}
			return hv.Call(args)[0].Interface()
		}
		f := &updateFold{
			eventType:    eventType,
			sample:       sample,
			valueType:    handlerType.Out(0),
			invoke:       invoke,
			keyExtractor: func(event any) any { return event },
		}
		f.recordSetter = func(r record.Record) { recHolder.rec = r }
		return f

	case numIn == 2 && numOut == 1:
		outType := handlerType.Out(0)
		if outType == reflect.TypeOf(Delta{}) {
			invoke := func(event any) Delta {
				return callWithRecord(event)[0].Interface().(Delta)
			}
			f := &countFold{
				eventType: eventType,
				sample:    sample,
				invoke:    invoke,
			}
			f.recordSetter = func(r record.Record) { recHolder.rec = r }
			return f
		}
		if outType == reflect.TypeOf(Edge{}) {
			invoke := func(event any) Edge {
				return callWithRecord(event)[0].Interface().(Edge)
			}
			f := &edgeFold{
				eventType: eventType,
				sample:    sample,
				invoke:    invoke,
			}
			f.recordSetter = func(r record.Record) { recHolder.rec = r }
			return f
		}

		invoke := func(event any) any {
			return callWithRecord(event)[0].Interface()
		}
		f := &setFold{
			eventType: eventType,
			sample:    sample,
			invoke:    invoke,
		}
		f.recordSetter = func(r record.Record) { recHolder.rec = r }
		return f
	}

	panic(fmt.Sprintf(
		"metaengine.OnRecord(%s): unsupported handler signature %v (see OnRecord docs)",
		eventType, handlerType,
	))
}
