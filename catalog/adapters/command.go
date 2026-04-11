package adapters

import (
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/command"
)

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

func AddCommandFromType[T command.Catalogable](
	b *CatalogBuilder,
	serviceID, cmdType string,
	meta command.CatalogMeta,
) {
	schema := catalog.SchemaFromType[T]()

	msg := catalog.Message{
		Kind:      catalog.CommandMessage,
		ID:        cmdType,
		Name:      meta.Name,
		Version:   meta.Version,
		Summary:   meta.Summary,
		Schema:    schema,
		Direction: catalog.Receives,
	}

	b.addMessageToService(serviceID, catalog.CommandMessage, msg)
}
