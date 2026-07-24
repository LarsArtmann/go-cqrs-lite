package metaengine

import (
	"errors"
	"fmt"
	"reflect"
)

// FoldKind classifies what a fold function does to the projection.
type FoldKind string

const (
	FoldInsert      FoldKind = "insert"
	FoldUpdate      FoldKind = "update"
	FoldRemove      FoldKind = "remove"
	FoldCount       FoldKind = "count"
	FoldEdge        FoldKind = "edge"
	FoldSet         FoldKind = "set"
	FoldSkip        FoldKind = "skip"
	FoldMultiInsert FoldKind = "multi_insert"
	FoldAppend      FoldKind = "append"
)

// Fold is a single event-to-projection mapping.
// The Kind field tells the planner which ADT operation this fold performs.
// Fold structs are created by the On constructor and should not be built by hand.
type Fold struct {
	EventType   string
	EventSample any
	Kind        FoldKind

	// keyType holds the key type K for FoldInsert/FoldSet,
	// or the value type V for FoldRemove/FoldUpdate (matched during derivation).
	keyType reflect.Type

	// valueType is the value type V for FoldInsert/FoldUpdate/FoldRemove.
	valueType reflect.Type

	insertHandler any // func(e E) (K, V)
	updateHandler any // func(e E, prev V) V
	keyExtractor  any // func(event any) any — auto-derived or set during Query construction
	countHandler  any // func(e E) Delta
	edgeHandler   any // func(e E) Edge
	setHandler    any // func(e E) K

	multiInsertHandler any // func(e E) MultiEntry
	appendHandler      any // func(e E) Append
}

func EventTypeName(sample any) string {
	t := reflect.TypeOf(sample)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return t.Name()
}

// removeSignal is the sentinel returned by Remove[V]().
// On classifies it as FoldRemove and uses valueType for projection matching.
type removeSignal struct {
	valueType reflect.Type
}

// Remove returns a sentinel that tells On to classify this fold as a deletion.
// The type parameter V must match the value type of the insert fold so the
// planner knows which projection to delete from:
//
//	metaengine.On(UserDeleted{}, metaengine.Remove[FindUserResult]())
//
// The key is auto-derived by scanning the event struct for fields matching
// the insert fold's key type.
func Remove[V any]() removeSignal {
	return removeSignal{valueType: reflect.TypeFor[V]()}
}

// On declares a fold: how an event of type E updates the query's projection.
// The handler's Go signature determines the ADT operation:
//
//	metaengine.On(Event{}, func(e Event) (Key, Value) { ... })   // Map insert
//	metaengine.On(Event{}, func(e Event, prev Value) Value { ... }) // Map update
//	metaengine.On(Event{}, func(e Event) Key { ... })            // Set add
//	metaengine.On(Event{}, func(e Event) metaengine.Delta { ... }) // Counter
//	metaengine.On(Event{}, func(e Event) metaengine.Edge { ... })  // Graph edge
//	metaengine.On(Event{}, metaengine.Remove[Value]())             // Delete
//	metaengine.On(Event{}, func(e Event) metaengine.Skip { ... })  // No-op
//
// On panics at init time if the handler signature is unclassifiable,
// following the MustCompile convention.
func On[E any](sample E, handler any) Fold {
	eventType := EventTypeName(sample)

	// Check sentinel types first.
	if rs, ok := handler.(removeSignal); ok {
		return Fold{
			EventType:   eventType,
			EventSample: sample,
			Kind:        FoldRemove,
			valueType:   rs.valueType,
		}
	}

	ht := reflect.TypeOf(handler)
	if ht == nil || ht.Kind() != reflect.Func {
		panic(fmt.Sprintf(
			"metaengine.On(%s): handler must be a function or Remove[V](), got %T",
			eventType, handler,
		))
	}

	if err := verifyEventParam[E](ht, eventType); err != nil {
		panic(err.Error())
	}

	numIn := ht.NumIn()
	numOut := ht.NumOut()

	switch {
	case numIn == 1 && numOut == 2:
		return Fold{
			EventType:     eventType,
			EventSample:   sample,
			Kind:          FoldInsert,
			keyType:       ht.Out(0),
			valueType:     ht.Out(1),
			insertHandler: handler,
		}

	case numIn == 2 && numOut == 1:
		return Fold{
			EventType:     eventType,
			EventSample:   sample,
			Kind:          FoldUpdate,
			valueType:     ht.Out(0),
			updateHandler: handler,
		}

	case numIn == 1 && numOut == 1:
		return classifySingleReturn(sample, eventType, ht.Out(0), handler)

	default:
		panic(fmt.Sprintf(
			"metaengine.On(%s): handler must have 1-2 params and 1-2 returns, got %d in / %d out",
			eventType, numIn, numOut,
		))
	}
}

func classifySingleReturn[E any](
	sample E,
	eventType string,
	outType reflect.Type,
	handler any,
) Fold {
	deltaType := reflect.TypeFor[Delta]()
	edgeType := reflect.TypeFor[Edge]()
	skipType := reflect.TypeFor[Skip]()
	multiEntryType := reflect.TypeFor[MultiEntry]()
	appendType := reflect.TypeFor[Append]()

	switch outType {
	case deltaType:
		return Fold{
			EventType:    eventType,
			EventSample:  sample,
			Kind:         FoldCount,
			countHandler: handler,
		}
	case edgeType:
		return Fold{
			EventType:   eventType,
			EventSample: sample,
			Kind:        FoldEdge,
			edgeHandler: handler,
		}
	case skipType:
		return Fold{
			EventType:   eventType,
			EventSample: sample,
			Kind:        FoldSkip,
		}
	case multiEntryType:
		return Fold{
			EventType:          eventType,
			EventSample:        sample,
			Kind:               FoldMultiInsert,
			multiInsertHandler: handler,
		}
	case appendType:
		return Fold{
			EventType:     eventType,
			EventSample:   sample,
			Kind:          FoldAppend,
			appendHandler: handler,
		}
	default:
		return Fold{
			EventType:   eventType,
			EventSample: sample,
			Kind:        FoldSet,
			keyType:     outType,
			setHandler:  handler,
		}
	}
}

func verifyEventParam[E any](ht reflect.Type, eventType string) error {
	var sample E

	expectedType := reflect.TypeOf(sample)
	if expectedType.Kind() == reflect.Pointer {
		expectedType = expectedType.Elem()
	}

	paramType := ht.In(0)
	if paramType.Kind() == reflect.Pointer {
		paramType = paramType.Elem()
	}

	if paramType != expectedType {
		return fmt.Errorf(
			"metaengine.On(%s): handler first param must be %s, got %s",
			eventType, expectedType, ht.In(0),
		)
	}

	return nil
}

// ── call helpers (used by Store.applyFold) ──

func (f *Fold) callInsert(event any) (any, any) {
	fn := reflect.ValueOf(f.insertHandler)
	results := fn.Call([]reflect.Value{reflect.ValueOf(event)})

	return results[0].Interface(), results[1].Interface()
}

func (f *Fold) callUpdate(event any, prev any) any {
	fn := reflect.ValueOf(f.updateHandler)

	args := []reflect.Value{reflect.ValueOf(event)}
	if prev != nil {
		args = append(args, reflect.ValueOf(prev))
	} else {
		args = append(args, reflect.Zero(fn.Type().In(1)))
	}

	return fn.Call(args)[0].Interface()
}

func (f *Fold) callKey(event any) any {
	if f.keyExtractor == nil {
		return nil
	}

	fn := reflect.ValueOf(f.keyExtractor)

	return fn.Call([]reflect.Value{reflect.ValueOf(event)})[0].Interface()
}

func (f *Fold) callCount(event any) Delta {
	fn := reflect.ValueOf(f.countHandler)

	return fn.Call([]reflect.Value{reflect.ValueOf(event)})[0].Interface().(Delta)
}

func (f *Fold) callEdge(event any) Edge {
	fn := reflect.ValueOf(f.edgeHandler)

	return fn.Call([]reflect.Value{reflect.ValueOf(event)})[0].Interface().(Edge)
}

func (f *Fold) callSet(event any) any {
	fn := reflect.ValueOf(f.setHandler)

	return fn.Call([]reflect.Value{reflect.ValueOf(event)})[0].Interface()
}

func (f *Fold) callMultiInsert(event any) MultiEntry {
	fn := reflect.ValueOf(f.multiInsertHandler)

	return fn.Call([]reflect.Value{reflect.ValueOf(event)})[0].Interface().(MultiEntry)
}

func (f *Fold) callAppend(event any) Append {
	fn := reflect.ValueOf(f.appendHandler)

	return fn.Call([]reflect.Value{reflect.ValueOf(event)})[0].Interface().(Append)
}

// ── ADT classification ──

func classifyADT(folds []Fold) (ADT, error) {
	hasInsert, hasSet, hasCount, hasEdge, hasMulti, hasAppend := false, false, false, false, false, false

	for _, f := range folds {
		switch f.Kind {
		case FoldInsert, FoldUpdate, FoldRemove:
			hasInsert = true
		case FoldSet:
			hasSet = true
		case FoldCount:
			hasCount = true
		case FoldEdge:
			hasEdge = true
		case FoldMultiInsert:
			hasMulti = true
		case FoldAppend:
			hasAppend = true
		}
	}

	switch {
	case hasEdge:
		return ADTGraph, nil
	case hasCount:
		return ADTCounter, nil
	case hasMulti:
		return ADTMultimap, nil
	case hasAppend:
		return ADTLog, nil
	case hasSet:
		return ADTSet, nil
	case hasInsert:
		return ADTMap, nil
	default:
		return "", errors.New("cannot infer ADT: no active folds (only skips)")
	}
}

// deriveKeys auto-generates keyExtractor closures for update and remove folds
// by matching the insert fold's key type against fields in the event struct.
func deriveKeys(folds []Fold) error {
	var keyType reflect.Type

	for _, f := range folds {
		if f.Kind == FoldInsert {
			keyType = f.keyType

			break
		}
	}

	if keyType == nil {
		return nil
	}

	for i := range folds {
		switch folds[i].Kind {
		case FoldUpdate, FoldRemove:
			if folds[i].keyExtractor != nil {
				continue
			}

			extractor, err := buildKeyExtractor(folds[i].EventSample, keyType)
			if err != nil {
				return fmt.Errorf("fold for %s: %w", folds[i].EventType, err)
			}

			folds[i].keyExtractor = extractor
		}
	}

	return nil
}

func buildKeyExtractor(eventSample any, keyType reflect.Type) (any, error) {
	t := reflect.TypeOf(eventSample)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	foundIdx := -1

	for i := range t.NumField() {
		if !t.Field(i).IsExported() {
			continue
		}

		if t.Field(i).Type == keyType {
			if foundIdx >= 0 {
				return nil, fmt.Errorf(
					"ambiguous key: multiple fields of type %s in %s (%s, %s)",
					keyType, t.Name(), t.Field(foundIdx).Name, t.Field(i).Name,
				)
			}

			foundIdx = i
		}
	}

	if foundIdx < 0 {
		return nil, fmt.Errorf("no field of type %s in %s", keyType, t.Name())
	}

	idx := foundIdx

	return func(event any) any {
		v := reflect.ValueOf(event)
		if v.Kind() == reflect.Pointer {
			v = v.Elem()
		}

		return v.Field(idx).Interface()
	}, nil
}
