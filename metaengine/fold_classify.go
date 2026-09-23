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
		case *edgeFold, *edgeRemoveFold:
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
		if keyed, ok := f.(extractorFold); ok {
			if err := ensureKeyExtractor(keyed, keyType); err != nil {
				return err
			}
		}
	}

	return nil
}

// extractorFold is the set of fold kinds whose event key is extracted from
// the event payload (update and remove folds).
type extractorFold interface {
	fold()
	EventType() string
	EventSample() any
	keyExtractorPtr() *func(event any) any
}

func (f *updateFold) keyExtractorPtr() *func(event any) any { return &f.keyExtractor }
func (f *removeFold) keyExtractorPtr() *func(event any) any { return &f.keyExtractor }

// ensureKeyExtractor lazily builds a fold's event-key extractor from its
// event sample, leaving an already-built extractor untouched.
func ensureKeyExtractor(f extractorFold, keyType reflect.Type) error {
	if *f.keyExtractorPtr() != nil {
		return nil
	}

	extractor, err := buildKeyExtractor(f.EventSample(), keyType)
	if err != nil {
		return fmt.Errorf("fold for %s: %w", f.EventType(), err)
	}

	*f.keyExtractorPtr() = extractor.(func(event any) any)

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
