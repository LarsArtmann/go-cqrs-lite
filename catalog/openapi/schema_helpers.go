package openapi

import (
	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

func schemaKey(msg catalog.Message) string {
	return string(msg.Kind) + "." + string(msg.ID)
}

func schemaToAny(s *catalog.Schema) any {
	result, err := schema.ToAny(s)
	if err != nil {
		return objectSchema()
	}

	return result
}

func objectSchema() map[string]string {
	return map[string]string{"type": string(schema.TypeObject)}
}
