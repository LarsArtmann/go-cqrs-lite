package metaengine

import (
	"fmt"
	"reflect"
)

// FoldKind classifies what a fold function does to the projection.
// Retained for diagnostics (hooks, logging); dispatch uses a type switch
// on the sealed Fold interface, not this string.
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
	FoldSpatial     FoldKind = "spatial"
)

// Fold is a sealed interface representing a single event-to-projection mapping.
// Each concrete implementation carries exactly one typed handler — there are
// no nil slots to accidentally invoke, eliminating the nil-panic class entirely.
//
// The concrete types (insertFold, updateFold, etc.) are unexported, making the
// union sealed: only the On/OnTyped constructors can create Fold values.
//
// Fold structs are created by the On constructor and should not be built by hand.
type Fold interface {
	fold() // sealed — unexported method prevents external implementations
	EventType() string
	EventSample() any
	Kind() FoldKind
}

// ── Concrete fold types ──
// Each type stores exactly its own typed handler as a pre-bound closure.
// The closure captures the reflect.Value once at construction time;
// the hot path calls the closure without per-event reflect.ValueOf(handler).

// insertFold: func(E) (K, V) → MapSet(collection, K, V)
type insertFold struct {
	eventType string
	sample    any
	keyType   reflect.Type
	valueType reflect.Type
	invoke    func(event any) (key, val any)
}

func (f *insertFold) fold()             {}
func (f *insertFold) EventType() string { return f.eventType }
func (f *insertFold) EventSample() any  { return f.sample }
func (f *insertFold) Kind() FoldKind    { return FoldInsert }

// updateFold: func(E, prev V) V → MapUpdate
type updateFold struct {
	eventType    string
	sample       any
	valueType    reflect.Type
	invoke       func(event, prev any) any
	keyExtractor func(event any) any
}

func (f *updateFold) fold()             {}
func (f *updateFold) EventType() string { return f.eventType }
func (f *updateFold) EventSample() any  { return f.sample }
func (f *updateFold) Kind() FoldKind    { return FoldUpdate }

// removeFold: key extraction from event → MapDelete
type removeFold struct {
	eventType    string
	sample       any
	valueType    reflect.Type
	keyExtractor func(event any) any
}

func (f *removeFold) fold()             {}
func (f *removeFold) EventType() string { return f.eventType }
func (f *removeFold) EventSample() any  { return f.sample }
func (f *removeFold) Kind() FoldKind    { return FoldRemove }

// countFold: func(E) Delta → CounterIncrement
type countFold struct {
	eventType string
	sample    any
	invoke    func(event any) Delta
}

func (f *countFold) fold()             {}
func (f *countFold) EventType() string { return f.eventType }
func (f *countFold) EventSample() any  { return f.sample }
func (f *countFold) Kind() FoldKind    { return FoldCount }

// edgeFold: func(E) Edge → GraphAddEdge
type edgeFold struct {
	eventType string
	sample    any
	invoke    func(event any) Edge
}

func (f *edgeFold) fold()             {}
func (f *edgeFold) EventType() string { return f.eventType }
func (f *edgeFold) EventSample() any  { return f.sample }
func (f *edgeFold) Kind() FoldKind    { return FoldEdge }

// setFold: func(E) K → SetAdd
type setFold struct {
	eventType string
	sample    any
	keyType   reflect.Type
	invoke    func(event any) any
}

func (f *setFold) fold()             {}
func (f *setFold) EventType() string { return f.eventType }
func (f *setFold) EventSample() any  { return f.sample }
func (f *setFold) Kind() FoldKind    { return FoldSet }

// multiInsertFold: func(E) MultiEntry → MultiAdd
type multiInsertFold struct {
	eventType string
	sample    any
	invoke    func(event any) MultiEntry
}

func (f *multiInsertFold) fold()             {}
func (f *multiInsertFold) EventType() string { return f.eventType }
func (f *multiInsertFold) EventSample() any  { return f.sample }
func (f *multiInsertFold) Kind() FoldKind    { return FoldMultiInsert }

// appendFold: func(E) Append → LogAppend
type appendFold struct {
	eventType string
	sample    any
	invoke    func(event any) Append
}

func (f *appendFold) fold()             {}
func (f *appendFold) EventType() string { return f.eventType }
func (f *appendFold) EventSample() any  { return f.sample }
func (f *appendFold) Kind() FoldKind    { return FoldAppend }

// vectorFold: func(E) Embedding → VectorInsert
type vectorFold struct {
	eventType string
	sample    any
	invoke    func(event any) Embedding
}

func (f *vectorFold) fold()             {}
func (f *vectorFold) EventType() string { return f.eventType }
func (f *vectorFold) EventSample() any  { return f.sample }
func (f *vectorFold) Kind() FoldKind    { return FoldVector }

// searchFold: func(E) IndexedText → SearchInsert
type searchFold struct {
	eventType string
	sample    any
	invoke    func(event any) IndexedText
}

func (f *searchFold) fold()             {}
func (f *searchFold) EventType() string { return f.eventType }
func (f *searchFold) EventSample() any  { return f.sample }
func (f *searchFold) Kind() FoldKind    { return FoldSearch }

// spatialFold: func(E) Point → SpatialInsert
type spatialFold struct {
	eventType string
	sample    any
	invoke    func(event any) Point
}

func (f *spatialFold) fold()             {}
func (f *spatialFold) EventType() string { return f.eventType }
func (f *spatialFold) EventSample() any  { return f.sample }
func (f *spatialFold) Kind() FoldKind    { return FoldSpatial }

// skipFold: func(E) Skip → no-op
type skipFold struct {
	eventType string
	sample    any
}

func (f *skipFold) fold()             {}
func (f *skipFold) EventType() string { return f.eventType }
func (f *skipFold) EventSample() any  { return f.sample }
func (f *skipFold) Kind() FoldKind    { return FoldSkip }

// ── Helpers ──

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

// reflectCall1 creates a pre-bound closure that calls a single-param,
// single-return reflect.Value and type-asserts the result to T.
// The reflect.Value is captured once at construction time; the hot path
// does not call reflect.ValueOf(handler) per event.
func reflectCall1[T any](hv reflect.Value) func(any) T {
	return func(event any) T {
		return hv.Call([]reflect.Value{reflect.ValueOf(event)})[0].Interface().(T)
	}
}

func onFold[E any](eventType string, sample E, handler any) Fold {
	if rs, ok := handler.(removeSignal); ok {
		return &removeFold{
			eventType: eventType,
			sample:    sample,
			valueType: rs.valueType,
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

	hv := reflect.ValueOf(handler)
	numIn := handlerType.NumIn()
	numOut := handlerType.NumOut()

	switch {
	case numIn == 1 && numOut == 2:
		invoke := func(event any) (any, any) {
			results := hv.Call([]reflect.Value{reflect.ValueOf(event)})

			return results[0].Interface(), results[1].Interface()
		}

		return &insertFold{
			eventType: eventType,
			sample:    sample,
			keyType:   handlerType.Out(0),
			valueType: handlerType.Out(1),
			invoke:    invoke,
		}

	case numIn == 2 && numOut == 1:
		invoke := func(event, prev any) any {
			args := []reflect.Value{reflect.ValueOf(event)}
			prevType := hv.Type().In(1)

			if prev != nil {
				args = append(args, reifyReflect(prev, prevType))
			} else {
				args = append(args, reflect.Zero(prevType))
			}

			return hv.Call(args)[0].Interface()
		}

		return &updateFold{
			eventType: eventType,
			sample:    sample,
			valueType: handlerType.Out(0),
			invoke:    invoke,
		}

	case numIn == 1 && numOut == 1:
		return classifySingleReturn(sample, eventType, handlerType.Out(0), hv)

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
	hv reflect.Value,
) Fold {
	switch outType {
	case reflect.TypeFor[Embedding]():
		return &vectorFold{eventType: eventType, sample: sample, invoke: reflectCall1[Embedding](hv)}

	case reflect.TypeFor[IndexedText]():
		return &searchFold{eventType: eventType, sample: sample, invoke: reflectCall1[IndexedText](hv)}

	case reflect.TypeFor[Point]():
		return &spatialFold{eventType: eventType, sample: sample, invoke: reflectCall1[Point](hv)}

	case reflect.TypeFor[Delta]():
		return &countFold{eventType: eventType, sample: sample, invoke: reflectCall1[Delta](hv)}

	case reflect.TypeFor[Edge]():
		return &edgeFold{eventType: eventType, sample: sample, invoke: reflectCall1[Edge](hv)}

	case reflect.TypeFor[Skip]():
		return &skipFold{eventType: eventType, sample: sample}

	case reflect.TypeFor[MultiEntry]():
		return &multiInsertFold{eventType: eventType, sample: sample, invoke: reflectCall1[MultiEntry](hv)}

	case reflect.TypeFor[Append]():
		return &appendFold{eventType: eventType, sample: sample, invoke: reflectCall1[Append](hv)}

	default:
		return &setFold{
			eventType: eventType,
			sample:    sample,
			keyType:   outType,
			invoke: func(event any) any {
				return hv.Call([]reflect.Value{reflect.ValueOf(event)})[0].Interface()
			},
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
