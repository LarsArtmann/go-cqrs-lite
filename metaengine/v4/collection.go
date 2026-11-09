package metaengine

import (
	"encoding/json/v2"
	"reflect"
)

// colResultInfo describes a collection result type.
type colResultInfo struct {
	itemsFieldIdx  int          // index of the []T field
	itemsElemType  reflect.Type // element type T
	cursorFieldIdx int          // index of the *Cursor field, or -1 if none
}

// collectionResultInfo inspects a result type R for collection characteristics.
// A collection result is a struct with at least one slice field ([]T).
// If a *Cursor field exists, it is used for pagination continuation.
func collectionResultInfo(t reflect.Type) (*colResultInfo, bool) {
	if t == nil || t.Kind() != reflect.Struct {
		return nil, false
	}

	info := &colResultInfo{cursorFieldIdx: -1}
	foundSlice := false

	cursorType := reflect.TypeFor[*Cursor]()

	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		if !foundSlice && f.Type.Kind() == reflect.Slice {
			info.itemsFieldIdx = i
			info.itemsElemType = f.Type.Elem()
			foundSlice = true
		}

		if info.cursorFieldIdx < 0 && f.Type == cursorType {
			info.cursorFieldIdx = i
		}
	}

	if !foundSlice {
		return nil, false
	}

	return info, true
}

func isCollectionResult[R any]() bool {
	var zero R

	_, ok := collectionResultInfo(reflect.TypeOf(zero))

	return ok
}

// reconstructCollection builds a typed collection result from []any returned
// by the engine. It fills in the slice field and optionally the cursor field.
func reconstructCollection[R any](result ScanResult, sortKeyFn func(any) any) R {
	var zero R

	t := reflect.TypeOf(zero)

	info, ok := collectionResultInfo(t)
	if !ok {
		return zero
	}

	items := result.Items

	slice := reflect.MakeSlice(reflect.SliceOf(info.itemsElemType), 0, len(items))

	var lastItem any

	for _, item := range items {
		if item == nil {
			continue
		}

		if jv, ok := item.(jsonValue); ok {
			elem := reflect.New(info.itemsElemType)

			if err := json.Unmarshal(jv, elem.Interface()); err == nil {
				slice = reflect.Append(slice, elem.Elem())
			}

			lastItem = item

			continue
		}

		val := reflect.ValueOf(item)
		if val.Type().ConvertibleTo(info.itemsElemType) {
			slice = reflect.Append(slice, val.Convert(info.itemsElemType))
		} else {
			slice = reflect.Append(slice, reifyReflect(item, info.itemsElemType))
		}

		lastItem = item
	}

	result2 := reflect.New(t).Elem()
	result2.Field(info.itemsFieldIdx).Set(slice)

	if result.HasMore && info.cursorFieldIdx >= 0 && lastItem != nil {
		cursorVal := lastItem
		if sortKeyFn != nil {
			cursorVal = sortKeyFn(lastItem)
		}

		cursor := &Cursor{Value: cursorVal}
		result2.Field(info.cursorFieldIdx).Set(reflect.ValueOf(cursor))
	}

	return result2.Interface().(R)
}
