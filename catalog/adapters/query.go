package adapters

import (
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/query"
)

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

func AddQueryFromType[T query.Catalogable](
	b *CatalogBuilder,
	serviceID, queryType string,
	meta query.CatalogMeta,
) {
	schema := catalog.SchemaFromType[T]()

	msg := catalog.Message{
		Kind:    catalog.QueryMessage,
		ID:      queryType,
		Name:    meta.Name,
		Version: meta.Version,
		Summary: meta.Summary,
		Schema:  schema,
	}

	b.addMessageToService(serviceID, catalog.QueryMessage, msg)
}
