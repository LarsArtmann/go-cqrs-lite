package adapters

import (
	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/core/command"
)

// AddCommand registers a command with a service, inferring schema from reflection.
func (b *CatalogBuilder) AddCommand(serviceID string, cmd command.Catalogable) {
	meta := cmd.CatalogInfo()
	msg := buildCommandMessageFromReflect(string(cmd.Type()), meta, cmd)
	b.addMessageToService(serviceID, catalog.CommandMessage, msg)
}

// AddCommandWithSchema registers a command with an explicit schema.
func (b *CatalogBuilder) AddCommandWithSchema(
	serviceID string,
	cmd command.Catalogable,
	schema *catalog.Schema,
) {
	meta := cmd.CatalogInfo()
	msg := buildCommandMessage(string(cmd.Type()), meta, schema)
	b.addMessageToService(serviceID, catalog.CommandMessage, msg)
}

// AddCommandFromType registers a command using generic type inference for schema generation.
func AddCommandFromType[T command.Catalogable](
	builder *CatalogBuilder,
	serviceID, cmdType string,
	meta command.CatalogMeta,
) {
	schema := catalog.SchemaFromType[T]()
	msg := buildCommandMessage(cmdType, meta, schema)
	builder.addMessageToService(serviceID, catalog.CommandMessage, msg)
}
