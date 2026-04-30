package adapters

import (
	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

func addMessagesToService(
	builder *CatalogBuilder,
	serviceID string,
	kind catalog.MessageKind,
	msgs []catalog.Message,
) {
	for _, msg := range msgs {
		builder.addMessageToService(serviceID, kind, msg)
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
	msgs := make([]catalog.Message, 0, len(dispatcher.CatalogEntries()))
	for cmdType, meta := range dispatcher.CatalogEntries() {
		msgs = append(msgs, catalog.Message{
			Kind:      catalog.CommandMessage,
			ID:        string(cmdType),
			Name:      meta.Name,
			Version:   meta.Version,
			Summary:   meta.Summary,
			Direction: catalog.Receives,
			Schema:    nil,
			Examples:  nil,
		})
	}

	addMessagesToService(builder, serviceID, catalog.CommandMessage, msgs)
}

// FromQueryDispatcher adds all catalog entries from a query dispatcher
// to the builder for the given service. Schemas must be added separately
// via AddQueryFromType[T] if needed.
func FromQueryDispatcher(
	builder *CatalogBuilder,
	serviceID string,
	dispatcher *query.Dispatcher,
) {
	msgs := make([]catalog.Message, 0, len(dispatcher.CatalogEntries()))
	for queryType, meta := range dispatcher.CatalogEntries() {
		msgs = append(msgs, catalog.Message{
			Kind:      catalog.QueryMessage,
			ID:        string(queryType),
			Name:      meta.Name,
			Version:   meta.Version,
			Summary:   meta.Summary,
			Direction: catalog.Receives,
			Schema:    nil,
			Examples:  nil,
		})
	}

	addMessagesToService(builder, serviceID, catalog.QueryMessage, msgs)
}
