package metaengine

import (
	"errors"
	"reflect"
)

// FoldKind classifies what a fold function does to the projection.
type FoldKind string

const (
	FoldInsert FoldKind = "insert"
	FoldUpdate FoldKind = "update"
	FoldRemove FoldKind = "remove"
	FoldCount  FoldKind = "count"
	FoldEdge   FoldKind = "edge"
	FoldSet    FoldKind = "set"
	FoldSkip   FoldKind = "skip"
)

// Fold is a single event-to-projection mapping.
type Fold struct {
	EventType   string
	EventSample any
	Kind        FoldKind

	insertHandler any
	updateHandler any
	keyExtractor  any
	countHandler  any
	edgeHandler   any
	setHandler    any
}

func EventTypeName(sample any) string {
	t := reflect.TypeOf(sample)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return t.Name()
}

// OnInsert registers a fold that creates a new (key, value) pair in a Map.
// The return type (K, V) tells the planner: ADT = Map<K, V>.
func OnInsert[E any, K any, V any](sample E, fn func(e E) (K, V)) Fold {
	return Fold{
		EventType:     EventTypeName(sample),
		EventSample:   sample,
		Kind:          FoldInsert,
		insertHandler: fn,
	}
}

// OnUpdate registers a fold that modifies an existing value (read-modify-write).
// keyFn extracts the key from the event so the engine knows which record to update.
func OnUpdate[E any, K any, V any](sample E, keyFn func(e E) K, fn func(e E, prev V) V) Fold {
	return Fold{
		EventType:     EventTypeName(sample),
		EventSample:   sample,
		Kind:          FoldUpdate,
		updateHandler: fn,
		keyExtractor:  keyFn,
	}
}

// OnRemove registers a fold that deletes a key from the projection.
// keyFn extracts the key from the event so the engine knows which record to delete.
func OnRemove[E any, K any](sample E, keyFn func(e E) K) Fold {
	return Fold{
		EventType:    EventTypeName(sample),
		EventSample:  sample,
		Kind:         FoldRemove,
		keyExtractor: keyFn,
	}
}

// OnCount registers a fold that adjusts a counter.
func OnCount[E any](sample E, fn func(e E) Delta) Fold {
	return Fold{
		EventType:    EventTypeName(sample),
		EventSample:  sample,
		Kind:         FoldCount,
		countHandler: fn,
	}
}

// OnCountTyped registers a fold that adjusts typed counters with branded keys.
// TypedDelta[K] uses a named string type for counter keys, making typos compile errors.
// The typed delta is converted to the generic Delta at registration time.
func OnCountTyped[E any, K ~string](sample E, fn func(e E) TypedDelta[K]) Fold {
	return Fold{
		EventType: EventTypeName(sample),
		EventSample: sample,
		Kind:       FoldCount,
		countHandler: func(e E) Delta {
			typed := fn(e)
			d := make(Delta, len(typed))
			for k, v := range typed {
				d[string(k)] = v
			}
			return d
		},
	}
}

// OnEdge registers a fold that adds a graph edge.
func OnEdge[E any](sample E, fn func(e E) Edge) Fold {
	return Fold{
		EventType:   EventTypeName(sample),
		EventSample: sample,
		Kind:        FoldEdge,
		edgeHandler: fn,
	}
}

// OnSet registers a fold that adds a key to a Set projection.
func OnSet[E any, K any](sample E, fn func(e E) K) Fold {
	return Fold{
		EventType:   EventTypeName(sample),
		EventSample: sample,
		Kind:        FoldSet,
		setHandler:  fn,
	}
}

// OnSkip registers a fold that explicitly ignores an event type.
func OnSkip[E any](sample E) Fold {
	return Fold{
		EventType:   EventTypeName(sample),
		EventSample: sample,
		Kind:        FoldSkip,
	}
}

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

func classifyADT(folds []Fold) (ADT, error) {
	hasInsert, hasSet, hasCount, hasEdge := false, false, false, false

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
		return "", errors.New("cannot infer ADT: no active folds (only skips)")
	}
}
