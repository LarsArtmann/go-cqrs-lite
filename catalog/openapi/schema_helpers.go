package openapi

import (
	"encoding/json"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

func schemaKey(msg catalog.Message) string {
	return string(msg.Kind) + "." + msg.ID
}

func schemaToAny(s *catalog.Schema) any {
	if s == nil {
		return objectSchema()
	}

	raw, err := json.Marshal(s)
	if err != nil {
		return objectSchema()
	}

	var result any

	err = json.Unmarshal(raw, &result)
	if err != nil {
		return objectSchema()
	}

	return result
}

func objectSchema() map[string]string {
	return map[string]string{"type": objectType}
}
