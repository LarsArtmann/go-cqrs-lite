package openapi

import (
	"github.com/larsartmann/go-cqrs-lite/catalog/v2"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2/schema"
)

func schemaKey(msg catalog.Message) string {
	return string(msg.Kind) + "." + string(msg.ID)
}

func schemaToAny(s *catalog.Schema) any {
	return schema.ToAny(s)
}

func objectSchema() map[string]string {
	return map[string]string{"type": string(schema.TypeObject)}
}
