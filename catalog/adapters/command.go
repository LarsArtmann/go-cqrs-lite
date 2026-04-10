package adapters

import (
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/command"
)

// AddCommand registers a command instance with the catalog builder.
// The command must implement command.Catalogable (embed command.CatalogCore).
// The schema is auto-extracted from the command's payload fields via reflection.
func (b *CatalogBuilder) AddCommand(serviceID string, cmd command.Catalogable) {
	meta := cmd.CatalogInfo()
	schema := catalog.SchemaFromReflect(reflect.TypeOf(cmd).Elem())

	msg := catalog.Message{
		Kind:      catalog.CommandMessage,
		ID:        string(cmd.Type()),
		Name:      meta.Name,
		Version:   meta.Version,
		Summary:   meta.Summary,
		Schema:    schema,
		Direction: catalog.Receives,
	}

	b.addMessageToService(serviceID, catalog.CommandMessage, msg)
}

// AddCommandWithSchema registers a command with an explicit schema override.
func (b *CatalogBuilder) AddCommandWithSchema(
	serviceID string,
	cmd command.Catalogable,
	schema *catalog.Schema,
) {
	meta := cmd.CatalogInfo()

	msg := catalog.Message{
		Kind:      catalog.CommandMessage,
		ID:        string(cmd.Type()),
		Name:      meta.Name,
		Version:   meta.Version,
		Summary:   meta.Summary,
		Schema:    schema,
		Direction: catalog.Receives,
	}

	b.addMessageToService(serviceID, catalog.CommandMessage, msg)
}
