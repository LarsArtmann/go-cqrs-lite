package adapters

import (
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/query"
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
	}
}

func buildCatalogMessageFromMeta(
	msgKind catalog.MessageKind,
	id string,
	meta catalogMetaProvider,
	schema *catalog.Schema,
	direction catalog.Direction,
) catalog.Message {
	return buildCatalogMessage(
		msgKind,
		id,
		meta.GetName(),
		meta.GetVersion(),
		meta.GetSummary(),
		schema,
		direction,
	)
}

type catalogMetaProvider interface {
	GetName() string
	GetVersion() string
	GetSummary() string
}

type commandMetaProvider struct {
	name    string
	version string
	summary string
}

func (m commandMetaProvider) GetName() string    { return m.name }
func (m commandMetaProvider) GetVersion() string { return m.version }
func (m commandMetaProvider) GetSummary() string { return m.summary }

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

type queryMetaProvider struct {
	name    string
	version string
	summary string
}

func (m queryMetaProvider) GetName() string    { return m.name }
func (m queryMetaProvider) GetVersion() string { return m.version }
func (m queryMetaProvider) GetSummary() string { return m.summary }

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

func buildQueryMessageFromReflect(
	id string,
	meta query.CatalogMeta,
	instance any,
) catalog.Message {
	schema := catalog.SchemaFromReflect(reflect.TypeOf(instance).Elem())
	return buildQueryMessage(id, meta, schema)
}

type eventMetaProvider struct {
	name    string
	version string
	summary string
}

func (m eventMetaProvider) GetName() string    { return m.name }
func (m eventMetaProvider) GetVersion() string { return m.version }
func (m eventMetaProvider) GetSummary() string { return m.summary }

func buildEventMessage(
	id string,
	meta event.EventCatalogMeta,
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

func buildEventMessageFromReflect(
	id string,
	meta event.EventCatalogMeta,
	instance any,
	direction catalog.Direction,
) catalog.Message {
	schema := catalog.SchemaFromReflect(reflect.TypeOf(instance).Elem())
	return buildEventMessage(id, meta, schema, direction)
}
