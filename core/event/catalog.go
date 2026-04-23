package event

import "github.com/larsartmann/go-cqrs-lite/core/pkg/id"

// EventCatalogMeta contains documentation metadata for auto-catalog generation.
type EventCatalogMeta struct {
	Name          string
	Version       string
	Summary       string
	AggregateType AggregateType
}

// EventCatalogable is implemented by events that want to be auto-documented
// by the catalog/adapters package.
type EventCatalogable interface {
	Event
	EventCatalogInfo() EventCatalogMeta
}

// EventCatalogCore combines event.Core with catalog metadata.
// Embed this struct in your event to make it auto-catalogable.
//
// Example:
//
//	type UserCreated struct {
//	    *EventCatalogCore
//	    Name  string `json:"name"`
//	    Email string `json:"email"`
//	}
type EventCatalogCore struct {
	*Core

	Meta EventCatalogMeta
}

// NewEventCatalogCore creates an EventCatalogCore with event metadata.
func NewEventCatalogCore(
	eventType Type,
	aggregateID id.AggregateID,
	aggregateType AggregateType,
	version int,
	payload []byte,
	meta EventCatalogMeta,
	opts ...Option,
) (*EventCatalogCore, error) {
	base, err := NewEvent(eventType, aggregateID, aggregateType, version, payload, opts...)
	if err != nil {
		return nil, err
	}

	return &EventCatalogCore{
		Core: base,
		Meta: meta,
	}, nil
}

// EventCatalogInfo returns the catalog metadata for this event.
func (c *EventCatalogCore) EventCatalogInfo() EventCatalogMeta {
	return c.Meta
}
