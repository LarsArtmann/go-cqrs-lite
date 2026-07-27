// Package schema provides reflection-based JSON Schema generation for the
// catalog exporters.
//
// It derives a [Schema] from Go types — structs, slices, maps, primitives, and
// time.Time — reading struct tags (json, doc/description, format, default, enum,
// nullable, deprecated, pattern). Results are cached per reflect.Type.
//
//	s := schema.FromType[UserCreated]()
//	jsonBytes, err := schema.ToJSON(s)
//	if err != nil { return err }
//	yamlBytes, err := schema.JSONToYAML(jsonBytes)
package schema
