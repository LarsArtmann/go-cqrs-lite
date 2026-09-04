//lint:exhaustruct This package creates partial schemas via helper functions.

package schema

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"maps"
	"reflect"
	"runtime"
	"slices"
	"sync"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

var ErrNilSchema = errorfamily.NewRejection("catalog.nil_schema", "schema is nil")

func FromType[T any]() *Schema {
	var zero T

	return fromReflect(reflect.TypeOf(zero))
}

func FromReflect(t reflect.Type) *Schema {
	return fromReflect(t)
}

// Clone returns a copy of the schema that is safe to mutate without
// affecting the package-level reflection cache. FromReflect and FromType
// return the SHARED cached *Schema for a type, so any caller that mutates
// the result (catalog message options such as WithParam append to
// Parameters) must clone first: without a clone, concurrent builders race
// on the cached schema and leak parameters into each other. The clone
// copies the mutable containers (Parameters, Properties, Required,
// Examples) with fresh ones sharing the read-only values; Type and Items
// are immutable after a build.
func (s *Schema) Clone() *Schema {
	if s == nil {
		return nil
	}

	clone := &Schema{
		Type:       s.Type,
		Items:      s.Items,
		Examples:   slices.Clone(s.Examples),
		Required:   slices.Clone(s.Required),
		Parameters: slices.Clone(s.Parameters),
	}

	if s.Properties != nil {
		clone.Properties = make(map[string]Property, len(s.Properties))
		maps.Copy(clone.Properties, s.Properties)
	}

	return clone
}

func ToJSON(s *Schema) ([]byte, error) {
	if s == nil {
		return nil, ErrNilSchema
	}

	//nolint:wrapcheck // MarshalIndent returns bytes, error from json.MarshalIndent
	return json.Marshal(
		s,
		json.Deterministic(true),
		jsontext.WithIndentPrefix(""),
		jsontext.WithIndent("  "),
	)
}

func ToAny(s *Schema) (any, error) {
	if s == nil {
		return nil, ErrNilSchema
	}

	raw, err := json.Marshal(s, json.Deterministic(true))
	if err != nil {
		return nil, errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.reflect.1",
			"marshal schema to JSON: %v",
			err,
		)
	}

	var result any

	err = json.Unmarshal(raw, &result, json.MatchCaseInsensitiveNames(true))
	if err != nil {
		return nil, errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.reflect.2",
			"unmarshal schema to any: %v",
			err,
		)
	}

	return result, nil
}

var (
	schemaCache sync.Map //nolint:gochecknoglobals // package-level reflection cache
	schemaBusy  sync.Map //nolint:gochecknoglobals // in-progress marker for cycle guard
)

func fromReflect(t reflect.Type) *Schema {
	if t == nil {
		return &Schema{Type: TypeNull}
	}

	if cached, ok := schemaCache.Load(t); ok {
		return cached.(*Schema) //nolint:forcetypeassert // cache only stores *Schema values
	}

	// Cycle guard: a self-referential type (e.g. a field of type *T inside
	// T) would otherwise recurse until the stack dies. The recursive
	// reference resolves to a plain-object placeholder ("opaque object");
	// the cached outer schema stays correct for every non-recursive field.
	// A concurrent same-type build also sees the busy marker and retries
	// on the completed cache entry instead of duplicating the build.
	if _, busy := schemaBusy.LoadOrStore(t, true); busy {
		return cyclePlaceholder(t)
	}

	s := buildSchema(t)
	schemaBusy.Delete(t)
	schemaCache.Store(t, s)

	return s
}

// cyclePlaceholder blocks until the in-flight build of t completes, then
// serves its cached schema. The build marks busy BEFORE recursing, so a
// re-entrant call from the building goroutine itself would deadlock here —
// recursion is therefore detected one frame earlier via LoadOrStore and
// never reaches this path from the builder.
func cyclePlaceholder(t reflect.Type) *Schema {
	for range 10000 {
		if cached, ok := schemaCache.Load(t); ok {
			return cached.(*Schema) //nolint:forcetypeassert // cache only stores *Schema values
		}

		runtime.Gosched()
	}

	return &Schema{Type: TypeObject}
}

func buildSchema(t reflect.Type) *Schema {
	if t == nil {
		return &Schema{Type: TypeNull}
	}

	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if isCollectionKind(t.Kind()) {
		return &Schema{
			Type:  TypeArray,
			Items: propertyFromReflect(t.Elem()),
		}
	}

	if t.Kind() == reflect.Map {
		return &Schema{
			Type: TypeObject,
			Properties: map[string]Property{
				"(key)":   *propertyFromReflect(t.Key()),
				"(value)": *propertyFromReflect(t.Elem()),
			},
		}
	}

	if t.Kind() != reflect.Struct {
		return &Schema{Type: goTypeToJSON(t.Kind())}
	}

	if t == reflect.TypeFor[time.Time]() {
		return &Schema{Type: TypeString}
	}

	return structSchema(t)
}

// maxEmbeddedDepth bounds the anonymous-field flattening walk. encoding/json
// promotes embedded struct fields to arbitrary depth; the schema flattener
// stops descending a few levels past anything sane instead of tracking a
// full conflict graph.
//
// Multi-embed name conflicts: if two embedded (or sibling) structs produce
// the same property name, the LAST field wins here — a plain map write.
// encoding/json instead DROPS all conflicting fields at equal depth. If a
// consumer relies on json's conflict-drop semantics, hand-write the schema
// (RegisterType with an explicit Schema) instead of relying on reflection.
const maxEmbeddedDepth = 8

func structSchema(t reflect.Type) *Schema {
	props := make(map[string]Property)

	var required []string

	var params []Parameter

	for field := range t.Fields() {
		if flatten, ft := flattenedEmbedded(field); flatten {
			flattenEmbedded(ft, props, &required, &params, 1)

			continue
		}

		name, prop, omit, include, param := fieldToProperty(field)
		if param.In != "" {
			params = append(params, param)
		}

		if !include {
			continue
		}

		props[name] = prop

		if !omit {
			required = append(required, name)
		}
	}

	s := &Schema{
		Type:       TypeObject,
		Properties: props,
		Required:   required,
	}

	if len(params) > 0 {
		s.Parameters = params
	}

	return s
}

func collectionSchema(t reflect.Type) *Property {
	if isCollectionKind(t.Kind()) {
		return &Property{
			Type:  TypeArray,
			Items: propertyFromReflect(t.Elem()),
		}
	}

	return &Property{Type: TypeObject}
}

func isCollectionKind(k reflect.Kind) bool {
	return k == reflect.Slice || k == reflect.Array
}

func propertyFromReflect(t reflect.Type) *Property {
	if t == nil {
		return &Property{Type: TypeNull}
	}

	if t.Kind() == reflect.Pointer {
		return propertyFromReflect(t.Elem())
	}

	if isCollectionKind(t.Kind()) {
		return collectionSchema(t)
	}

	if t.Kind() == reflect.Map {
		return &Property{Type: TypeObject}
	}

	if t.Kind() == reflect.Struct {
		if t == reflect.TypeFor[time.Time]() {
			return &Property{Type: TypeString, Format: "date-time"}
		}

		s := fromReflect(t)

		return &Property{
			Type:       s.Type,
			Properties: s.Properties,
			Required:   s.Required,
		}
	}

	return &Property{Type: goTypeToJSON(t.Kind())}
}

func goTypeToJSON(k reflect.Kind) Type {
	//nolint:exhaustive // default handles remaining kinds
	switch k {
	case reflect.String:
		return TypeString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return TypeInteger
	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr:
		return TypeInteger
	case reflect.Float32, reflect.Float64:
		return TypeNumber
	case reflect.Bool:
		return TypeBoolean
	case reflect.Interface:
		return TypeObject
	case reflect.Complex64, reflect.Complex128:
		return TypeString
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return TypeString
	case reflect.Invalid:
		return TypeNull
	default:
		return TypeString
	}
}
