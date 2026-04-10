package adapters

import (
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/event"
)

// AddEvent registers an event with the catalog builder.
// The event must implement event.EventCatalogable (embed event.EventCatalogCore).
// The schema is auto-extracted from the event's payload fields via reflection.
func (b *CatalogBuilder) AddEvent(serviceID string, evt event.EventCatalogable) {
	meta := evt.EventCatalogInfo()
	schema := extractEventSchema(evt)

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

// AddEventWithDirection registers an event with an explicit direction override.
func (b *CatalogBuilder) AddEventWithDirection(
	serviceID string,
	evt event.EventCatalogable,
	direction catalog.Direction,
) {
	meta := evt.EventCatalogInfo()
	schema := extractEventSchema(evt)

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

func extractEventSchema(evt event.EventCatalogable) *catalog.Schema {
	t := reflect.TypeOf(evt).Elem()

	return schemaFromReflect(t)
}
