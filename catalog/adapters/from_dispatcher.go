package adapters

import (
	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/core/command"
)

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
