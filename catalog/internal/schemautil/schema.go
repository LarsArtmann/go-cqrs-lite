package schemautil

import (
	"encoding/json"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

// SchemaToAny converts a catalog.Schema to a generic map for JSON serialization.
// Returns a fallback object schema on nil or error.
func SchemaToAny(s *catalog.Schema) any {
	if s == nil {
		return ObjectSchema()
	}

	raw, err := json.Marshal(s)
	if err != nil {
		return ObjectSchema()
	}

	var result any

	err = json.Unmarshal(raw, &result)
	if err != nil {
		return ObjectSchema()
	}

	return result
}

// ObjectSchema returns a minimal JSON Schema for an object type.
func ObjectSchema() map[string]string {
	return map[string]string{"type": string(catalog.TypeObject)}
}
