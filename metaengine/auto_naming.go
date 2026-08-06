package metaengine

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// autoInsertByType is the non-generic core of AutoInsert. It builds an insert
// fold from reflect.Type values, enabling convention-based inference (ADR-0116).
func autoInsertByType(eventType, resultType reflect.Type, keyField string) Fold {
	var sample any = reflect.Zero(eventType).Interface()

	keyIdx, err := findField(eventType, keyField)
	if err != nil {
		panic(fmt.Sprintf("AutoInsert: %s (event %s)", err, eventType.Name()))
	}

	keyType := eventType.Field(keyIdx).Type
	mappings := matchFields(eventType, resultType)
	stamps := computeRecordStamps(resultType, mappings)

	recHolder := &struct{ rec record.Record }{}

	invoke := func(event any) (key, val any) {
		eVal := reflect.ValueOf(event)
		k := eVal.Field(keyIdx).Interface()

		result := reflect.New(resultType).Elem()

		for _, m := range mappings {
			result.Field(m.dstIdx).Set(eVal.Field(m.srcIdx))
		}

		applyRecordStamps(result, stamps, recHolder.rec)

		return k, result.Interface()
	}

	f := &insertFold{
		eventType: EventTypeName(sample),
		sample:    sample,
		keyType:   keyType,
		valueType: resultType,
		invoke:    invoke,
	}
	f.recordSetter = func(r record.Record) { recHolder.rec = r }
	return f
}

// autoUpdateByType is the non-generic core of AutoUpdate.
func autoUpdateByType(eventType, resultType reflect.Type, keyField string) Fold {
	var sample any = reflect.Zero(eventType).Interface()

	keyIdx, err := findField(eventType, keyField)
	if err != nil {
		panic(fmt.Sprintf("AutoUpdate: %s (event %s)", err, eventType.Name()))
	}

	mappings := matchFields(eventType, resultType)
	stamps := computeRecordStamps(resultType, mappings)

	recHolder := &struct{ rec record.Record }{}

	invoke := func(event, prev any) any {
		eVal := reflect.ValueOf(event)

		result := reflect.New(resultType).Elem()
		if prev != nil {
			prevVal := reflect.ValueOf(prev)
			if prevVal.Type() == resultType {
				result.Set(prevVal)
			}
		}

		for _, m := range mappings {
			srcVal := eVal.Field(m.srcIdx)
			if !srcVal.IsZero() {
				result.Field(m.dstIdx).Set(srcVal)
			}
		}

		applyRecordStamps(result, stamps, recHolder.rec)

		return result.Interface()
	}

	f := &updateFold{
		eventType: EventTypeName(sample),
		sample:    sample,
		valueType: resultType,
		invoke:    invoke,
		keyExtractor: func(event any) any {
			return reflect.ValueOf(event).Field(keyIdx).Interface()
		},
	}
	f.recordSetter = func(r record.Record) { recHolder.rec = r }
	return f
}

// autoDeleteByType is the non-generic core of AutoDelete.
func autoDeleteByType(eventType reflect.Type, keyField string) Fold {
	var sample any = reflect.Zero(eventType).Interface()

	keyIdx, err := findField(eventType, keyField)
	if err != nil {
		panic(fmt.Sprintf("AutoDelete: %s (event %s)", err, eventType.Name()))
	}

	return &removeFold{
		eventType: EventTypeName(sample),
		sample:    sample,
		keyExtractor: func(event any) any {
			return reflect.ValueOf(event).Field(keyIdx).Interface()
		},
	}
}

// AutoCRUDByConvention generates insert, update, and delete folds by scanning
// event type names for Created/Updated/Deleted suffixes (ADR-0116 Layer 1).
// This eliminates the need to specify C/U/D type parameters explicitly.
//
// The function inspects each sample's Go type name:
//   - "*Created" suffix → insert fold (e.g. UserCreated)
//   - "*Updated" suffix → update fold (e.g. UserUpdated)
//   - "*Deleted" suffix → delete fold (e.g. UserDeleted)
//
// Example:
//
//	folds, err := metaengine.AutoCRUDByConvention[UserView]("ID",
//	    UserCreated{}, UserUpdated{}, UserDeleted{})
//	q := metaengine.Query[GetUser, UserView]("users", folds...)
//
// Returns an error if no Created sample is found (insert is the minimum
// requirement), or if multiple samples match the same suffix.
func AutoCRUDByConvention[R any](keyField string, samples ...any) ([]Fold, error) {
	resultType := reflect.TypeFor[R]()

	var createdType, updatedType, deletedType reflect.Type

	for _, s := range samples {
		t := reflect.TypeOf(s)
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}

		name := t.Name()

		switch {
		case strings.HasSuffix(name, "Created"):
			if createdType != nil {
				return nil, fmt.Errorf(
					"AutoCRUDByConvention: multiple Created types: %s and %s",
					createdType.Name(), name,
				)
			}
			createdType = t

		case strings.HasSuffix(name, "Updated"):
			if updatedType != nil {
				return nil, fmt.Errorf(
					"AutoCRUDByConvention: multiple Updated types: %s and %s",
					updatedType.Name(), name,
				)
			}
			updatedType = t

		case strings.HasSuffix(name, "Deleted"):
			if deletedType != nil {
				return nil, fmt.Errorf(
					"AutoCRUDByConvention: multiple Deleted types: %s and %s",
					deletedType.Name(), name,
				)
			}
			deletedType = t

		default:
			return nil, fmt.Errorf(
				"AutoCRUDByConvention: type %s does not match *Created/*Updated/*Deleted suffix",
				name,
			)
		}
	}

	if createdType == nil {
		return nil, fmt.Errorf(
			"AutoCRUDByConvention: no *Created sample provided (at least one is required)",
		)
	}

	var folds []Fold

	folds = append(folds, autoInsertByType(createdType, resultType, keyField))

	if updatedType != nil {
		folds = append(folds, autoUpdateByType(updatedType, resultType, keyField))
	}

	if deletedType != nil {
		folds = append(folds, autoDeleteByType(deletedType, keyField))
	}

	return folds, nil
}
