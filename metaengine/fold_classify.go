package metaengine

import (
	"errors"
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

// eventTypesForFolds returns the set of event types the given folds react to.
func eventTypesForFolds(folds []Fold) []string {
	seen := make(map[string]struct{}, len(folds))

	for _, f := range folds {
		if f.Kind != FoldSkip {
			seen[f.EventType] = struct{}{}
		}
	}

	result := make([]string, 0, len(seen))
	for t := range seen {
		result = append(result, t)
	}

	return result
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
