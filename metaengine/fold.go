package metaengine

import (
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
	FoldVector      FoldKind = "vector"
	FoldSearch      FoldKind = "search"
)

// Fold is a single event-to-projection mapping.
// The Kind field tells the planner which ADT operation this fold performs.
// Fold structs are created by the On constructor and should not be built by hand.
type Fold struct {
	EventType   string
	EventSample any
	Kind        FoldKind

	keyType   reflect.Type
	valueType reflect.Type

	insertHandler any // func(e E) (K, V)
	updateHandler any // func(e E, prev V) V
	keyExtractor  any // func(event any) any
	countHandler  any // func(e E) Delta
	edgeHandler   any // func(e E) Edge
	setHandler    any // func(e E) K

	multiInsertHandler any // func(e E) MultiEntry
	appendHandler      any // func(e E) Append
	vectorHandler      any // func(e E) Embedding
	searchHandler      any // func(e E) IndexedText
}

func EventTypeName(sample any) string {
	t := reflect.TypeOf(sample)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return t.Name()
}

// removeSignal is the sentinel returned by Remove[V]().
type removeSignal struct {
	valueType reflect.Type
}

// Remove returns a sentinel that tells On to classify this fold as a deletion.
func Remove[V any]() removeSignal {
	return removeSignal{valueType: reflect.TypeFor[V]()}
}

// On declares a fold: how an event of type E updates the query's projection.
// The event type string is derived from the sample's Go type name. The handler's
// Go signature determines the ADT operation:
//
//	metaengine.On(Event{}, func(e Event) (Key, Value) { ... })   // Map insert
//	metaengine.On(Event{}, func(e Event, prev Value) Value { ... }) // Map update
//	metaengine.On(Event{}, func(e Event) Key { ... })            // Set add
//	metaengine.On(Event{}, func(e Event) metaengine.Delta { ... }) // Counter
//	metaengine.On(Event{}, func(e Event) metaengine.Edge { ... })  // Graph edge
//	metaengine.On(Event{}, metaengine.Remove[Value]())             // Delete
//
// Use OnTyped when the event type string must differ from the Go type name
// (e.g. binding to a CQRS event.Type() string like "user.created").
func On[E any](sample E, handler any) Fold {
	return onFold(EventTypeName(sample), sample, handler)
}

// OnTyped is like On but binds the fold to an explicit event-type string
// instead of deriving it from the sample's Go type name. This is the bridge for
// CQRS events whose wire type string (event.Type(), e.g. "user.created") does
// not match the payload struct name (e.g. "UserCreated"). store.Apply(ctx,
// eventType, payload) matches folds by this string.
func OnTyped[E any](eventType string, sample E, handler any) Fold {
	return onFold(eventType, sample, handler)
}

func onFold[E any](eventType string, sample E, handler any) Fold {
	if rs, ok := handler.(removeSignal); ok {
		return Fold{
			EventType:   eventType,
			EventSample: sample,
			Kind:        FoldRemove,
			valueType:   rs.valueType,
		}
	}

	handlerType := reflect.TypeOf(handler)
	if handlerType == nil || handlerType.Kind() != reflect.Func {
		panic(fmt.Sprintf(
			"metaengine.On(%s): handler must be a function or Remove[V](), got %T",
			eventType, handler,
		))
	}

	if err := verifyEventParam[E](handlerType, eventType); err != nil {
		panic(err.Error())
	}

	numIn := handlerType.NumIn()
	numOut := handlerType.NumOut()

	switch {
	case numIn == 1 && numOut == 2:
		return Fold{
			EventType:     eventType,
			EventSample:   sample,
			Kind:          FoldInsert,
			keyType:       handlerType.Out(0),
			valueType:     handlerType.Out(1),
			insertHandler: handler,
		}

	case numIn == 2 && numOut == 1:
		return Fold{
			EventType:     eventType,
			EventSample:   sample,
			Kind:          FoldUpdate,
			valueType:     handlerType.Out(0),
			updateHandler: handler,
		}

	case numIn == 1 && numOut == 1:
		return classifySingleReturn(sample, eventType, handlerType.Out(0), handler)

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
	embeddingType := reflect.TypeFor[Embedding]()
	indexedTextType := reflect.TypeFor[IndexedText]()

	switch outType {
	case embeddingType:
		return Fold{
			EventType:     eventType,
			EventSample:   sample,
			Kind:          FoldVector,
			vectorHandler: handler,
		}
	case indexedTextType:
		return Fold{
			EventType:     eventType,
			EventSample:   sample,
			Kind:          FoldSearch,
			searchHandler: handler,
		}
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

func verifyEventParam[E any](handlerType reflect.Type, eventType string) error {
	var sample E

	expectedType := reflect.TypeOf(sample)
	if expectedType.Kind() == reflect.Pointer {
		expectedType = expectedType.Elem()
	}

	paramType := handlerType.In(0)
	if paramType.Kind() == reflect.Pointer {
		paramType = paramType.Elem()
	}

	if paramType != expectedType {
		return fmt.Errorf("metaengine.On(%s): %w %s, got %s",
			eventType, errInvalidEventType, expectedType, handlerType.In(0))
	}

	return nil
}
