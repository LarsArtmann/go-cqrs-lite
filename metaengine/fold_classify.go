package metaengine

import (
	"encoding/json/v2"
	"fmt"
	"reflect"
)

// call helpers (used by Store.applyFold)

func (f *Fold) callInsert(event any) (any, any) {
	fn := reflect.ValueOf(f.insertHandler)
	results := fn.Call([]reflect.Value{reflect.ValueOf(event)})

	return results[0].Interface(), results[1].Interface()
}

func (f *Fold) callUpdate(event any, prev any) any {
	fn := reflect.ValueOf(f.updateHandler)
	prevType := fn.Type().In(1)

	args := []reflect.Value{reflect.ValueOf(event)}
	if prev != nil {
		args = append(args, reifyReflect(prev, prevType))
	} else {
		args = append(args, reflect.Zero(prevType))
	}

	return fn.Call(args)[0].Interface()
}

// reifyReflect converts value into a reflect.Value assignable to target.
//
// Memory engines store and return typed Go values directly, so value is
// already assignable and is returned as-is (no JSON round-trip, no alloc).
// SQL engines JSON-encode on write and decode into any on read, producing
// map[string]any for structs — which is not assignable to a typed parameter
// and would panic inside a reflect.Call. reifyReflect rebuilds the typed
// value via JSON round-trip, mirroring reify[R] (reify.go) for the reflect
// call sites that do not have a static type parameter.
//
// Reification cannot fail for values an engine itself wrote (they are valid
// JSON of exactly target), so the round-trip is lossless. A marshal/unmarshal
// failure (only possible for externally-corrupted data, or for a raw cursor
// scalar passed where a struct is expected) falls back to the zero value of
// target rather than panicking.
func reifyReflect(value any, target reflect.Type) reflect.Value {
	if rt := reflect.TypeOf(value); rt != nil && rt.AssignableTo(target) {
		return reflect.ValueOf(value)
	}

	b, err := json.Marshal(value)
	if err != nil {
		return reflect.Zero(target)
	}

	v := reflect.New(target)

	if err := json.Unmarshal(b, v.Interface()); err != nil {
		return reflect.Zero(target)
	}

	return v.Elem()
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
		case FoldSkip:
			// Skips do not influence ADT selection.
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
		return "", errCannotInferADT
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
		case FoldInsert, FoldCount, FoldEdge, FoldSet, FoldSkip, FoldMultiInsert, FoldAppend:
			// Only update/remove folds need a derived key extractor.
		}
	}

	return nil
}

func buildKeyExtractor(eventSample any, keyType reflect.Type) (any, error) {
	t := derefType(eventSample)

	foundIdx := -1

	for i := range t.NumField() {
		if !t.Field(i).IsExported() {
			continue
		}

		if t.Field(i).Type == keyType {
			if foundIdx >= 0 {
				return nil, fmt.Errorf("%w: type %s in %s (%s, %s)",
					errAmbiguousKey, keyType, t.Name(), t.Field(foundIdx).Name, t.Field(i).Name)
			}

			foundIdx = i
		}
	}

	if foundIdx < 0 {
		return nil, fmt.Errorf("%w: type %s in %s", errNoKeyField, keyType, t.Name())
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
