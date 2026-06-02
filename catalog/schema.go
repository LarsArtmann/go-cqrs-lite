package catalog

import (
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/catalog/v2/schema"
)

func SchemaFromType[T any]() *Schema {
	return schema.FromType[T]()
}

func SchemaFromReflect(t reflect.Type) *Schema {
	return schema.FromReflect(t)
}

func SchemaToJSON(s *Schema) ([]byte, error) {
	return schema.ToJSON(s) //nolint:wrapcheck // thin delegation to sub-package
}

func SchemaToAny(s *Schema) (any, error) {
	return schema.ToAny(s)
}
