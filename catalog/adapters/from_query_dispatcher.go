package adapters

import (
	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

// FromQueryDispatcher adds all catalog entries from a query dispatcher
// to the builder for the given service. Schemas must be added separately
// via AddQueryFromType[T] if needed.
func FromQueryDispatcher(
	builder *CatalogBuilder,
	serviceID string,
	dispatcher *query.Dispatcher,
) {
	for queryType, meta := range dispatcher.CatalogEntries() {
		msg := catalog.Message{
			Kind:      catalog.QueryMessage,
			ID:        string(queryType),
			Name:      meta.Name,
			Version:   meta.Version,
			Summary:   meta.Summary,
			Direction: catalog.Receives,
			Schema:    nil,
			Examples:  nil,
		}

		builder.addMessageToService(serviceID, catalog.QueryMessage, msg)
	}
}
