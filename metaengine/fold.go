package metaengine

import (
	"fmt"
	"reflect"
)

// FoldKind classifies what a fold function does to the projection.
type FoldKind string

const (
	FoldInsert FoldKind = "insert" // event creates a new (key, value) entry
	FoldUpdate FoldKind = "update" // event modifies an existing value (read-modify-write)
	FoldRemove FoldKind = "remove" // event deletes a key from the projection
	FoldCount  FoldKind = "count"  // event adjusts a counter
	FoldEdge   FoldKind = "edge"   // event adds a graph edge
	FoldSet    FoldKind = "set"    // event adds a key to a set (membership)
	FoldSkip   FoldKind = "skip"   // event does not apply to this projection
)

// Fold is a single event-to-projection mapping.
// Exactly one of the handler fields is non-nil, determined by FoldKind.
type Fold struct {
	EventType   string
	EventSample any // zero-value instance of the event type for reflection
	Kind        FoldKind

	// For FoldInsert: func(event E) (key K, value V)
	insertHandler any
	// For FoldUpdate: func(event E, prev V) V
	updateHandler any
	// For FoldRemove: no handler needed, just the event type
	// For FoldCount: func(event E) Delta
	countHandler any
	// For FoldEdge: func(event E) Edge
	edgeHandler any
	// For FoldSet: func(event E) K (returns just the key)
	setHandler any
}

// EventTypeName extracts the type name from an event sample.
func EventTypeName(sample any) string {
	t := reflect.TypeOf(sample)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

// OnInsert registers a fold that creates a new (key, value) pair in a Map projection.
// The return type (K, V) tells the planner: ADT = Map<K, V>.
func OnInsert[E any, K any, V any](sample E, fn func(e E) (K, V)) Fold {
	return Fold{
		EventType:    EventTypeName(sample),
		EventSample:  sample,
		Kind:         FoldInsert,
		insertHandler: fn,
	}
}

// OnUpdate registers a fold that modifies an existing value (read-modify-write).
// The previous value is loaded from the projection, the function returns the new value.
func OnUpdate[E any, V any](sample E, fn func(e E, prev V) V) Fold {
	return Fold{
		EventType:    EventTypeName(sample),
		EventSample:  sample,
		Kind:         FoldUpdate,
		updateHandler: fn,
	}
}

// OnRemove registers a fold that deletes a key from the projection.
// The generic type V tells the planner which projection type to remove from.
func OnRemove[E any, V any](sample E) Fold {
	return Fold{
		EventType:   EventTypeName(sample),
		EventSample: sample,
		Kind:        FoldRemove,
	}
}

// OnCount registers a fold that adjusts a counter.
// The Delta return type tells the planner: ADT = Counter.
func OnCount[E any](sample E, fn func(e E) Delta) Fold {
	return Fold{
		EventType:   EventTypeName(sample),
		EventSample: sample,
		Kind:        FoldCount,
		countHandler: fn,
	}
}

// OnEdge registers a fold that adds a graph edge.
// The Edge return type tells the planner: ADT = Graph.
func OnEdge[E any](sample E, fn func(e E) Edge) Fold {
	return Fold{
		EventType:   EventTypeName(sample),
		EventSample: sample,
		Kind:        FoldEdge,
		edgeHandler: fn,
	}
}

// OnSet registers a fold that adds a key to a Set projection.
// Returning just a key (no value) tells the planner: ADT = Set<K>.
func OnSet[E any, K any](sample E, fn func(e E) K) Fold {
	return Fold{
		EventType:   EventTypeName(sample),
		EventSample: sample,
		Kind:        FoldSet,
		setHandler:  fn,
	}
}

// OnSkip registers a fold that explicitly ignores an event type.
// Use this to document that a projection intentionally skips certain events.
func OnSkip[E any](sample E) Fold {
	return Fold{
		EventType:   EventTypeName(sample),
		EventSample: sample,
		Kind:        FoldSkip,
	}
}

// callInsert invokes an insert handler with a decoded event payload.
func (f *Fold) callInsert(event any) (key any, value any) {
	fn := reflect.ValueOf(f.insertHandler)
	results := fn.Call([]reflect.Value{reflect.ValueOf(event)})
	return results[0].Interface(), results[1].Interface()
}

// callUpdate invokes an update handler with a decoded event payload and previous value.
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

// callCount invokes a counter handler.
func (f *Fold) callCount(event any) Delta {
	fn := reflect.ValueOf(f.countHandler)
	return fn.Call([]reflect.Value{reflect.ValueOf(event)})[0].Interface().(Delta)
}

// callEdge invokes an edge handler.
func (f *Fold) callEdge(event any) Edge {
	fn := reflect.ValueOf(f.edgeHandler)
	return fn.Call([]reflect.Value{reflect.ValueOf(event)})[0].Interface().(Edge)
}

// callSet invokes a set handler.
func (f *Fold) callSet(event any) any {
	fn := reflect.ValueOf(f.setHandler)
	return fn.Call([]reflect.Value{reflect.ValueOf(event)})[0].Interface()
}

// classifyADT determines the ADT from the fold kinds in a query.
func classifyADT(folds []Fold) (ADT, error) {
	hasInsert := false
	hasSet := false
	hasCount := false
	hasEdge := false
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
		}
	}
	switch {
	case hasEdge:
		return ADTGraph, nil
	case hasCount:
		return ADTCounter, nil
	case hasSet:
		return ADTSet, nil
	case hasInsert:
		return ADTMap, nil
	default:
		return "", fmt.Errorf("cannot infer ADT: no active folds (only skips)")
	}
}
