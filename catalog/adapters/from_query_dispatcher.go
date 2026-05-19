package adapters

import (
	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

// FromCommandDispatcher adds all catalog entries from a command dispatcher
// to the builder for the given service. Schemas must be added separately
// via catalog.Command[T]() if needed.
//
// Deprecated: Use the zero-cost catalog API with catalog.Command[T]()
// instead of extracting from dispatcher catalog entries.
func FromCommandDispatcher(
	builder *CatalogBuilder,
	serviceID string,
	dispatcher *command.Dispatcher,
) {
	reg := builder.builder.Registry()

	for cmdType, meta := range dispatcher.CatalogEntries() {
		reg.AddCommand(serviceID, catalog.Message{
			Kind:      catalog.CommandMessage,
			ID:        string(cmdType),
			Name:      meta.Name,
			Version:   meta.Version,
			Summary:   meta.Summary,
			Direction: catalog.Receives,
		})
	}
}

// FromQueryDispatcher adds all catalog entries from a query dispatcher
// to the builder for the given service. Schemas must be added separately
// via catalog.Query[T]() if needed.
//
// Deprecated: Use the zero-cost catalog API with catalog.Query[T]()
// instead of extracting from dispatcher catalog entries.
func FromQueryDispatcher(
	builder *CatalogBuilder,
	serviceID string,
	dispatcher *query.Dispatcher,
) {
	reg := builder.builder.Registry()

	for queryType, meta := range dispatcher.CatalogEntries() {
		reg.AddQuery(serviceID, catalog.Message{
			Kind:      catalog.QueryMessage,
			ID:        string(queryType),
			Name:      meta.Name,
			Version:   meta.Version,
			Summary:   meta.Summary,
			Direction: catalog.Receives,
		})
	}
}
