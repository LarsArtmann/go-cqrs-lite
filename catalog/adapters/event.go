package adapters

import (
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/core/event"
)

func (b *CatalogBuilder) AddEvent(serviceID string, evt event.Catalogable) {
	meta := evt.CatalogInfo()
	schema := catalog.SchemaFromReflect(reflect.TypeOf(evt).Elem())
	msg := buildEventMessage(string(evt.Type()), meta, schema, catalog.Sends)
	b.addMessageToService(serviceID, catalog.EventMessage, msg)
}

func (b *CatalogBuilder) AddEventWithDirection(
	serviceID string,
	evt event.Catalogable,
	direction catalog.Direction,
) {
	meta := evt.CatalogInfo()
	schema := catalog.SchemaFromReflect(reflect.TypeOf(evt).Elem())
	msg := buildEventMessage(string(evt.Type()), meta, schema, direction)
	b.addMessageToService(serviceID, catalog.EventMessage, msg)
}

func AddEventFromType[T event.Catalogable](
	builder *CatalogBuilder,
	serviceID, eventType string,
	meta event.CatalogMeta,
	direction catalog.Direction,
) {
	schema := catalog.SchemaFromType[T]()
	msg := buildEventMessage(eventType, meta, schema, direction)
	builder.addMessageToService(serviceID, catalog.EventMessage, msg)
}
