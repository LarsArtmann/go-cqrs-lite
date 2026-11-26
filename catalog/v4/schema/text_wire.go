package schema

import (
	"encoding"
	"reflect"
)

// textWireType maps a struct that marshals itself as text to the JSON type
// of its single value field. A branded ID (go-branded-id's cbid.ID pattern)
// is such a struct: an unexported value behind MarshalText, a bare string or
// number on the wire — never the empty object its fields would suggest.
func textWireType(t reflect.Type) (Type, bool) {
	if t.NumField() != 1 || !reflect.PointerTo(t).Implements(reflect.TypeFor[encoding.TextMarshaler]()) {
		return "", false
	}

	wireType := goTypeToJSON(t.Field(0).Type.Kind())
	isScalar := wireType == TypeString || wireType == TypeInteger ||
		wireType == TypeNumber || wireType == TypeBoolean

	return wireType, isScalar
}
