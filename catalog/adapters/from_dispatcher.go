package adapters

import (
	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

// fromDispatcher[T] adds catalog entries from a dispatcher to the builder.
func fromDispatcher[T any](
	builder *CatalogBuilder,
	serviceID string,
	dispatcher interface{ CatalogEntries() map[T]struct{ Name, Version, Summary string } },
	msgKind catalog.MessageKind,
) {
	for msgType, meta := range dispatcher.CatalogEntries() {
		msg := catalog.Message{
			Kind:      msgKind,
			ID:        string(msgType.(string)),
			Name:      meta.Name,
			Version:   meta.Version,
			Summary:   meta.Summary,
			Direction: catalog.Receives,
			Schema:    nil,
			Examples:  nil,
		}

		builder.addMessageToService(serviceID, msgKind, msg)
	}
}

// FromCommandDispatcher adds all catalog entries from a command dispatcher
// to the builder for the given service. Schemas must be added separately
// via AddCommandFromType[T] if needed.
func FromCommandDispatcher(
	builder *CatalogBuilder,
	serviceID string,
	dispatcher *command.Dispatcher,
) {
	for cmdType, meta := range dispatcher.CatalogEntries() {
		msg := catalog.Message{
			Kind:      catalog.CommandMessage,
			ID:        string(cmdType),
			Name:      meta.Name,
			Version:   meta.Version,
			Summary:   meta.Summary,
			Direction: catalog.Receives,
			Schema:    nil,
			Examples:  nil,
		}

		builder.addMessageToService(serviceID, catalog.CommandMessage, msg)
	}
}

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
