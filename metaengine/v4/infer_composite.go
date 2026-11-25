package metaengine

import (
	"fmt"
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// keyDetection holds the result of key-field auto-detection from the query
// input type and Created event type. When composite is true, the key spans
// multiple fields and keyType is a dynamic struct created via reflect.StructOf.
type keyDetection struct {
	fields       []string // field names in the Created event
	composite    bool     // true when multiple fields form the key
	keyType      reflect.Type
	eventIndices []int // field indices in the Created event
}

// detectKeyFields auto-detects key fields from the query input type Q against
// the Created event type. Strategies (in priority order):
//
//  1. Single non-meta input field with an unambiguous type match in Created.
//  2. Multiple non-meta fields where one is named "ID" → "ID" is the key.
//  3. Multiple non-meta fields, all with unambiguous type matches, none named
//     "ID" → composite key.
//  4. Fallback to a field named "ID" in the Created event.
func detectKeyFields(queryType, createdType reflect.Type) (keyDetection, error) {
	inputFields := exportedNonMetaTypeFields(queryType)

	type fieldMatch struct {
		name string
		idx  int
	}

	var matches []fieldMatch

	seenCreatedIdx := make(map[int]bool)

	for _, in := range inputFields {
		// Skip fields that look like filter bounds (MinScore, MaxScore, etc.)
		// — they are filter candidates, not key candidates.
		if _, _, isFilter := inferFilterOp(in.Name); isFilter {
			continue
		}

		found := -1
		ambiguous := false

		for i := range createdType.NumField() {
			f := createdType.Field(i)
			if !f.IsExported() {
				continue
			}

			if f.Type == in.Type {
				if found >= 0 || seenCreatedIdx[i] {
					ambiguous = true
				}

				found = i
			}
		}

		if !ambiguous && found >= 0 && !seenCreatedIdx[found] {
			seenCreatedIdx[found] = true

			matches = append(matches, fieldMatch{
				name: createdType.Field(found).Name,
				idx:  found,
			})
		}
	}

	switch len(matches) {
	case 0:
		return detectIDFallback(createdType)

	case 1:
		return keyDetection{
			fields:       []string{matches[0].name},
			keyType:      createdType.Field(matches[0].idx).Type,
			eventIndices: []int{matches[0].idx},
		}, nil

	default:
		// If any matched field is named "ID", use it as sole key.
		for _, m := range matches {
			if m.name == "ID" {
				return keyDetection{
					fields:       []string{"ID"},
					keyType:      createdType.Field(m.idx).Type,
					eventIndices: []int{m.idx},
				}, nil
			}
		}

		// Composite key across all matched fields.
		fields := make([]string, len(matches))
		structFields := make([]reflect.StructField, len(matches))
		indices := make([]int, len(matches))

		for i, m := range matches {
			cf := createdType.Field(m.idx)
			fields[i] = cf.Name
			indices[i] = m.idx
			structFields[i] = reflect.StructField{Name: cf.Name, Type: cf.Type}
		}

		return keyDetection{
			fields:       fields,
			composite:    true,
			keyType:      reflect.StructOf(structFields),
			eventIndices: indices,
		}, nil
	}
}

// detectIDFallback returns a single-key detection for a field named "ID".
func detectIDFallback(createdType reflect.Type) (keyDetection, error) {
	for i := range createdType.NumField() {
		f := createdType.Field(i)
		if f.IsExported() && f.Name == "ID" {
			return keyDetection{
				fields:       []string{"ID"},
				keyType:      f.Type,
				eventIndices: []int{i},
			}, nil
		}
	}

	return keyDetection{}, fmt.Errorf(
		"infer: cannot auto-detect key field in %s (no unambiguous type matches and no ID field)",
		createdType.Name(),
	)
}

// buildCompositeKey constructs a composite key value from an event or input
// struct by extracting the fields at det.eventIndices.
func buildCompositeKey(val reflect.Value, det keyDetection) any {
	key := reflect.New(det.keyType).Elem()

	for i, idx := range det.eventIndices {
		key.Field(i).Set(val.Field(idx))
	}

	return key.Interface()
}

// generateCompositeFolds builds insert/update/delete folds that use a composite
// key. The key is a dynamic struct with one field per key component, built from
// the event via buildCompositeKey.
func generateCompositeFolds(
	c conventionClassification,
	resultType reflect.Type,
	det keyDetection,
) ([]Fold, error) {
	valueType := collectionElementType(resultType)

	var folds []Fold

	insert := compositeInsertFold(c.created, valueType, det)
	folds = append(folds, insert)

	for _, eventType := range nonNilTypes(c.updated, c.deleted) {
		eventDet, err := keyDetectionFor(eventType, det)
		if err != nil {
			return nil, err
		}

		if eventType == c.updated {
			folds = append(folds, compositeUpdateFold(eventType, valueType, eventDet))
		} else {
			folds = append(folds, compositeDeleteFold(eventType, eventDet))
		}
	}

	return folds, nil
}

func nonNilTypes(types ...reflect.Type) []reflect.Type {
	var out []reflect.Type

	for _, t := range types {
		if t != nil {
			out = append(out, t)
		}
	}

	return out
}

// keyDetectionFor re-resolves the composite key fields by NAME for one event
// type. The Created sample's field offsets say nothing about the Updated or
// Deleted structs: a different layout silently builds the wrong key, a
// missing field panics at apply time and poisons the collection.
func keyDetectionFor(eventType reflect.Type, det keyDetection) (keyDetection, error) {
	indices := make([]int, len(det.fields))

	for i, name := range det.fields {
		f, ok := eventType.FieldByName(name)
		if !ok {
			return det, fmt.Errorf("infer: %s has no key field %q", eventType.Name(), name)
		}

		if want := det.keyType.Field(i).Type; f.Type != want {
			return det, fmt.Errorf("infer: key field %s.%s is %s, want %s as declared by the Created event",
				eventType.Name(), name, f.Type, want)
		}

		indices[i] = f.Index[0]
	}

	det.eventIndices = indices

	return det, nil
}

func compositeInsertFold(
	createdType, valueType reflect.Type,
	det keyDetection,
) Fold {
	//art-dupl:accept insert/update folds share the sample+mappings prologue by design
	sample := reflect.Zero(createdType).Interface()
	mappings := matchFields(createdType, valueType)
	stamps := computeRecordStamps(valueType, mappings)

	invoke := func(rec record.Record, event any) (key, val any) {
		eVal := reflect.ValueOf(event)
		k := buildCompositeKey(eVal, det)

		result := reflect.New(valueType).Elem()

		for _, m := range mappings {
			result.Field(m.dstIdx).Set(fieldValue(eVal, m.srcPath))
		}

		applyRecordStamps(result, stamps, rec)

		return k, result.Interface()
	}

	f := &insertFold{
		eventType: EventTypeName(sample),
		sample:    sample,
		keyType:   det.keyType,
		valueType: valueType,
		invoke:    invoke,
	}

	return f
}

func compositeUpdateFold(
	updatedType, valueType reflect.Type,
	det keyDetection,
) Fold {
	//art-dupl:accept insert/update folds share the sample+mappings prologue by design
	sample := reflect.Zero(updatedType).Interface()
	mappings := matchFields(updatedType, valueType)
	stamps := computeRecordStamps(valueType, mappings)

	invoke := func(rec record.Record, event, prev any) any {
		eVal := reflect.ValueOf(event)

		result := reflect.New(valueType).Elem()
		if prev != nil {
			prevVal := reflect.ValueOf(prev)
			if prevVal.Type() == valueType {
				result.Set(prevVal)
			}
		}

		for _, m := range mappings {
			srcVal := fieldValue(eVal, m.srcPath)
			if !srcVal.IsZero() {
				result.Field(m.dstIdx).Set(srcVal)
			}
		}

		applyRecordStamps(result, stamps, rec)

		return result.Interface()
	}

	f := &updateFold{
		eventType: EventTypeName(sample),
		sample:    sample,
		valueType: valueType,
		invoke:    invoke,
		keyExtractor: func(event any) any {
			return buildCompositeKey(reflect.ValueOf(event), det)
		},
	}

	return f
}

func compositeDeleteFold(deletedType reflect.Type, det keyDetection) Fold {
	sample := reflect.Zero(deletedType).Interface()

	return &removeFold{
		eventType: EventTypeName(sample),
		sample:    sample,
		keyExtractor: func(event any) any {
			return buildCompositeKey(reflect.ValueOf(event), det)
		},
	}
}
