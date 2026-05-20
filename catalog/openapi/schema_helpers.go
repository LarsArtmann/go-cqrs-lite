package openapi

import (
	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/internal/schemautil"
)

func schemaKey(msg catalog.Message) string {
	return string(msg.Kind) + "." + string(msg.ID)
}

func schemaToAny(s *catalog.Schema) any {
	return schemautil.SchemaToAny(s)
}

func objectSchema() map[string]string {
	return schemautil.ObjectSchema()
}
