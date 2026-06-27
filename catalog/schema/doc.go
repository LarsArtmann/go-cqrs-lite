// Package schema provides reflection-based JSON Schema generation for the
// catalog exporters.
//
// It derives a [Schema] from Go types — structs, slices, maps, primitives, and
// time.Time — reading struct tags (json, doc/description, format, default, enum,
// nullable, deprecated, pattern). Results are cached per reflect.Type.
//
//	s := schema.FromType[UserCreated]()
//	jsonBytes, _ := schema.ToJSON(s)
//	yamlBytes, _ := schema.JSONToYAML(jsonBytes)
package schema
