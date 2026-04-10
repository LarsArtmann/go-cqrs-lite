package adapters

import (
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/query"
)

// AddQuery registers a query with the catalog builder.
// The query must implement query.Catalogable (embed query.CatalogCore).
// The schema is auto-extracted from the query's request fields via reflection.
func (b *CatalogBuilder) AddQuery(serviceID string, qry query.Catalogable) {
	meta := qry.CatalogInfo()
	schema := catalog.SchemaFromReflect(reflect.TypeOf(qry).Elem())

	msg := catalog.Message{
		Kind:    catalog.QueryMessage,
		ID:      string(qry.Type()),
		Name:    meta.Name,
		Version: meta.Version,
		Summary: meta.Summary,
		Schema:  schema,
	}

	b.addMessageToService(serviceID, catalog.QueryMessage, msg)
}
