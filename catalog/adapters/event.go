package adapters

import (
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/event"
)

func (b *CatalogBuilder) AddEvent(serviceID string, evt event.EventCatalogable) {
	meta := evt.EventCatalogInfo()
	schema := catalog.SchemaFromReflect(reflect.TypeOf(evt).Elem())

	msg := catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        string(evt.Type()),
		Name:      meta.Name,
		Version:   meta.Version,
		Summary:   meta.Summary,
		Schema:    schema,
		Direction: catalog.Sends,
	}

	b.addMessageToService(serviceID, catalog.EventMessage, msg)
}

func (b *CatalogBuilder) AddEventWithDirection(
	serviceID string,
	evt event.EventCatalogable,
	direction catalog.Direction,
) {
	meta := evt.EventCatalogInfo()
	schema := catalog.SchemaFromReflect(reflect.TypeOf(evt).Elem())

	msg := catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        string(evt.Type()),
		Name:      meta.Name,
		Version:   meta.Version,
		Summary:   meta.Summary,
		Schema:    schema,
		Direction: direction,
	}

	b.addMessageToService(serviceID, catalog.EventMessage, msg)
}

func AddEventFromType[T event.EventCatalogable](
	b *CatalogBuilder,
	serviceID, eventType string,
	meta event.EventCatalogMeta,
	direction catalog.Direction,
) {
	schema := catalog.SchemaFromType[T]()

	msg := catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        eventType,
		Name:      meta.Name,
		Version:   meta.Version,
		Summary:   meta.Summary,
		Schema:    schema,
		Direction: direction,
	}

	b.addMessageToService(serviceID, catalog.EventMessage, msg)
}
