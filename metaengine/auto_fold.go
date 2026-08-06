package metaengine

import (
	"fmt"
	"reflect"
)

// fieldMapping pairs a source field index (in the event struct) with a
// destination field index (in the result struct). Pre-computed at fold
// construction time so the hot path avoids map lookups.
type fieldMapping struct {
	srcIdx int
	dstIdx int
}

// findField returns the index of a named exported field in a struct type,
// or an error if the field doesn't exist or is unexported.
func findField(t reflect.Type, name string) (int, error) {
	for i := range t.NumField() {
		f := t.Field(i)
		if f.Name == name {
			if !f.IsExported() {
				return -1, fmt.Errorf("field %q is unexported in %s", name, t.Name())
			}

			return i, nil
		}
	}

	return -1, fmt.Errorf("field %q not found in %s", name, t.Name())
}

// matchFields finds all exported fields in srcType whose names match exported
// fields in dstType, with compatible (assignable) types. Returns a slice of
// fieldMapping values, pre-sorted by source index.
func matchFields(srcType, dstType reflect.Type) []fieldMapping {
	var mappings []fieldMapping

	for i := range srcType.NumField() {
		srcField := srcType.Field(i)
		if !srcField.IsExported() {
			continue
		}

		for j := range dstType.NumField() {
			dstField := dstType.Field(j)
			if !dstField.IsExported() {
				continue
			}

			if srcField.Name == dstField.Name && srcField.TypeAssignable(dstField.Type) {
				mappings = append(mappings, fieldMapping{srcIdx: i, dstIdx: j})

				break
			}
		}
	}

	return mappings
}

// AutoInsert creates an insert fold that maps event fields to the result type
// by name. The key is extracted from the named field in the event.
//
// Example:
//
//	type UserCreated struct { ID string; Name string; Email string }
//	type UserView    struct { ID string; Name string; Email string }
//
//	metaengine.AutoInsert[UserCreated, UserView]("ID")
//
// This is equivalent to the manual fold:
//
//	metaengine.On(UserCreated{}, func(e UserCreated) (string, UserView) {
//	    return e.ID, UserView{ID: e.ID, Name: e.Name, Email: e.Email}
//	})
//
// Field matching: all exported fields in E whose names match exported fields in R
// with assignable types are copied. Fields in R not present in E are left zero-valued.
func AutoInsert[E any, R any](keyField string) Fold {
	var sample E

	eventType := reflect.TypeOf(sample)
	resultType := reflect.TypeFor[R]()

	keyIdx, err := findField(eventType, keyField)
	if err != nil {
		panic(fmt.Sprintf("AutoInsert: %s (event %s)", err, eventType.Name()))
	}

	mappings := matchFields(eventType, resultType)

	handler := func(e E) (any, R) {
		eVal := reflect.ValueOf(e)
		key := eVal.Field(keyIdx).Interface()

		var result R
		resultVal := reflect.ValueOf(&result).Elem()

		for _, m := range mappings {
			resultVal.Field(m.dstIdx).Set(eVal.Field(m.srcIdx))
		}

		return key, result
	}

	return On(sample, handler)
}

// AutoDelete creates a delete fold that removes the entry by extracting the key
// from the named field in the event.
//
// Example:
//
//	type UserDeleted struct { ID string }
//
//	metaengine.AutoDelete[UserDeleted]("ID")
//
// This generates a removeFold with a keyExtractor that reads the ID field.
func AutoDelete[E any](keyField string) Fold {
	var sample E

	eventType := reflect.TypeOf(sample)

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

// AutoUpdate creates an update fold that merges non-zero event fields into the
// existing result value. Only fields present in both E and R are merged; fields
// that are zero-valued in the event are skipped (partial update semantics).
//
// Example:
//
//	type UserUpdated struct { ID string; Name string; Email string }
//	type UserView    struct { ID string; Name string; Email string; CreatedAt string }
//
//	metaengine.AutoUpdate[UserUpdated, UserView]("ID")
//
// This is equivalent to the manual fold:
//
//	metaengine.On(UserUpdated{}, func(e UserUpdated, prev UserView) UserView {
//	    if prev.ID == "" { prev.ID = e.ID }
//	    if e.Name != "" { prev.Name = e.Name }
//	    if e.Email != "" { prev.Email = e.Email }
//	    return prev
//	})
func AutoUpdate[E any, R any](keyField string) Fold {
	var sample E

	eventType := reflect.TypeOf(sample)
	resultType := reflect.TypeFor[R]()

	_, err := findField(eventType, keyField)
	if err != nil {
		panic(fmt.Sprintf("AutoUpdate: %s (event %s)", err, eventType.Name()))
	}

	mappings := matchFields(eventType, resultType)

	handler := func(e E, prev R) R {
		eVal := reflect.ValueOf(e)
		prevVal := reflect.ValueOf(&prev).Elem()

		for _, m := range mappings {
			srcVal := eVal.Field(m.srcIdx)
			if !srcVal.IsZero() {
				prevVal.Field(m.dstIdx).Set(srcVal)
			}
		}

		return prev
	}

	return On(sample, handler)
}

// AutoCRUD generates insert, update, and delete folds for a standard CRUD
// entity lifecycle. This is the convenience entry point for the 80% of
// projections that are simple CRUD-shaped (ADR-0116).
//
// Type parameters:
//   - C: the "created" event type (e.g. UserCreated)
//   - U: the "updated" event type (e.g. UserUpdated)
//   - D: the "deleted" event type (e.g. UserDeleted)
//   - R: the result/view type (e.g. UserView)
//
// Example:
//
//	folds := metaengine.AutoCRUD[UserCreated, UserUpdated, UserDeleted, UserView]("ID")
//	q := metaengine.Query[GetUser, UserView]("users", folds...)
func AutoCRUD[C, U, D, R any](keyField string) []Fold {
	return []Fold{
		AutoInsert[C, R](keyField),
		AutoUpdate[U, R](keyField),
		AutoDelete[D](keyField),
	}
}
