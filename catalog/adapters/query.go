package adapters

import (
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

// AddQuery registers a query with a service, inferring schema from reflection.
func (b *CatalogBuilder) AddQuery(serviceID string, qry query.Catalogable) {
	meta := qry.CatalogInfo()
	schema := catalog.SchemaFromReflect(reflect.TypeOf(qry).Elem())
	msg := buildQueryMessage(string(qry.Type()), meta, schema)
	b.addMessageToService(serviceID, catalog.QueryMessage, msg)
}

// AddQueryFromType registers a query using generic type inference for schema generation.
func AddQueryFromType[T query.Catalogable](
	builder *CatalogBuilder,
	serviceID, queryType string,
	meta query.CatalogMeta,
) {
	schema := catalog.SchemaFromType[T]()
	msg := buildQueryMessage(queryType, meta, schema)
	builder.addMessageToService(serviceID, catalog.QueryMessage, msg)
}
