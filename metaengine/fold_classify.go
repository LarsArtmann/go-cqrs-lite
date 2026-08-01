package metaengine

import (
	"fmt"
	"reflect"
)

// ── ADT classification ──

func classifyADT(folds []Fold) (ADT, error) {
	hasInsert, hasSet, hasCount, hasEdge, hasMulti, hasAppend, hasVector, hasSearch, hasSpatial := false, false, false, false, false, false, false, false, false

	for _, f := range folds {
		switch f.(type) {
		case *insertFold, *updateFold, *removeFold:
			hasInsert = true
		case *setFold:
			hasSet = true
		case *countFold:
			hasCount = true
		case *edgeFold:
			hasEdge = true
		case *multiInsertFold:
			hasMulti = true
		case *appendFold:
			hasAppend = true
		case *vectorFold:
			hasVector = true
		case *searchFold:
			hasSearch = true
		case *spatialFold:
			hasSpatial = true
		case *skipFold:
			// Skips do not influence ADT selection.
		}
	}

	switch {
	case hasVector:
		return ADTVector, nil
	case hasSearch:
		return ADTSearch, nil
	case hasSpatial:
		return ADTSpatial, nil
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
		if ins, ok := f.(*insertFold); ok {
			keyType = ins.keyType

			break
		}
	}

	if keyType == nil {
		return nil
	}

	for _, f := range folds {
		switch fold := f.(type) {
		case *updateFold:
			if fold.keyExtractor != nil {
				continue
			}

			extractor, err := buildKeyExtractor(fold.EventSample(), keyType)
			if err != nil {
				return fmt.Errorf("fold for %s: %w", fold.EventType(), err)
			}

			fold.keyExtractor = extractor

		case *removeFold:
			if fold.keyExtractor != nil {
				continue
			}

			extractor, err := buildKeyExtractor(fold.EventSample(), keyType)
			if err != nil {
				return fmt.Errorf("fold for %s: %w", fold.EventType(), err)
			}

			fold.keyExtractor = extractor
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
