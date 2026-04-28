package adapters

import (
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

func buildCatalogMessage(
	msgKind catalog.MessageKind,
	id, name, version, summary string,
	schema *catalog.Schema,
	direction catalog.Direction,
) catalog.Message {
	return catalog.Message{
		Kind:      msgKind,
		ID:        id,
		Name:      name,
		Version:   version,
		Summary:   summary,
		Schema:    schema,
		Direction: direction,
		Examples:  nil,
	}
}

func buildCommandMessage(
	id string,
	meta command.CatalogMeta,
	schema *catalog.Schema,
) catalog.Message {
	return buildCatalogMessage(
		catalog.CommandMessage,
		id,
		meta.Name,
		meta.Version,
		meta.Summary,
		schema,
		catalog.Receives,
	)
}

func buildCommandMessageFromReflect(
	id string,
	meta command.CatalogMeta,
	instance any,
) catalog.Message {
	schema := catalog.SchemaFromReflect(reflect.TypeOf(instance).Elem())

	return buildCommandMessage(id, meta, schema)
}

func buildQueryMessage(
	id string,
	meta query.CatalogMeta,
	schema *catalog.Schema,
) catalog.Message {
	return buildCatalogMessage(
		catalog.QueryMessage,
		id,
		meta.Name,
		meta.Version,
		meta.Summary,
		schema,
		catalog.Receives,
	)
}

func buildEventMessage(
	id string,
	meta event.CatalogMeta,
	schema *catalog.Schema,
	direction catalog.Direction,
) catalog.Message {
	return buildCatalogMessage(
		catalog.EventMessage,
		id,
		meta.Name,
		meta.Version,
		meta.Summary,
		schema,
		direction,
	)
}
