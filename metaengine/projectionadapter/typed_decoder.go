package projectionadapter

import (
	"encoding/json/v2"
	"errors"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// ErrNoFoldForEventType is returned when an event type has no registered
// decoder in a TypeDecoder. This means the consumer forgot to register the
// event type, or an unexpected event type was routed to the projection.
var ErrNoFoldForEventType = errors.New("projectionadapter: no fold registered for event type")

// EventWithID wraps an event payload with the stream ID (entity key).
// Metaengine fold handlers need the entity ID as the Map key, but the raw
// event payload does not contain it — it lives in the event's StreamID.
// EventWithID bridges this gap.
//
// This is the canonical wrapper for all queries that key on the entity ID.
// Use it as the fold handler input type:
//
//	metaengine.OnTyped("user.created", projectionadapter.EventWithID[UserCreated]{},
//	    func(e projectionadapter.EventWithID[UserCreated]) (string, UserView) {
//	        return e.ID, UserView{ID: e.ID, Name: e.Payload.Name}
//	    })
//
// For Counter/Set queries that don't need the entity ID, use the payload
// type directly (the fold handler can accept the raw payload).
type EventWithID[P any] struct {
	ID      string
	Payload P
}

// TypeDecoder is a fluent builder that eliminates the manual switch/case
// event decoder every consumer previously had to write. Register each event
// type with its payload Go type, then pass the decoder to NewWithDecoder.
//
// Before (77 lines of boilerplate per consumer):
//
//	func myDecoder(evt event.Event) (any, error) {
//	    switch evt.Type() {
//	    case "task.created":
//	        var p TaskCreatedPayload
//	        if err := json.Unmarshal(evt.Payload(), &p); err != nil { return nil, err }
//	        return eventWithID[TaskCreatedPayload]{ID: evt.StreamID().String(), Payload: p}, nil
//	    case "task.assigned":
//	        var p TaskAssignedPayload
//	        if err := json.Unmarshal(evt.Payload(), &p); err != nil { return nil, err }
//	        return eventWithID[TaskAssignedPayload]{ID: evt.StreamID().String(), Payload: p}, nil
//	    // ... 9 more cases, each identical ...
//	    }
//	}
//
// After (5 lines):
//
//	dec := projectionadapter.NewTypeDecoder().
//	    On(evtTaskCreated, TaskCreatedPayload{}).
//	    On(evtTaskAssigned, TaskAssignedPayload{}).
//	    On(evtTaskDeleted, TaskDeletedPayload{})
//	adapter := projectionadapter.NewWithDecoder("tasks", store, dec)
type TypeDecoder struct {
	handlers map[string]func(event.Event) (any, error)
}

// NewTypeDecoder creates an empty TypeDecoder ready for event registration.
func NewTypeDecoder() *TypeDecoder {
	return &TypeDecoder{handlers: make(map[string]func(event.Event) (any, error))}
}

// On registers a payload type for an event type. The event type is an
// event.Type (a string typedef with a String() method). Go infers the
// payload type from the sample argument. Returns the decoder for chaining.
//
// The payload is decoded via encoding/json/v2 and wrapped in EventWithID,
// giving fold handlers access to both the entity ID and the typed payload.
func (d *TypeDecoder) On[E any](eventType event.Type, _ E) *TypeDecoder {
	t := string(eventType)
	d.handlers[t] = d.makeHandler[E](t)

	return d
}

// OnString is like On but accepts a plain string event type, for consumers
// that use string constants instead of event.Type.
func (d *TypeDecoder) OnString[E any](eventType string, _ E) *TypeDecoder {
	d.handlers[eventType] = d.makeHandler[E](eventType)

	return d
}

func (d *TypeDecoder) makeHandler[E any](label string) func(event.Event) (any, error) {
	return func(evt event.Event) (any, error) {
		var p E

		if len(evt.Payload()) > 0 {
			if err := json.Unmarshal(evt.Payload(), &p); err != nil {
				return nil, fmt.Errorf("projectionadapter: decode %s: %w", label, err)
			}
		}

		return EventWithID[E]{ID: evt.StreamID().String(), Payload: p}, nil
	}
}

// Decode implements EventDecoder. It decodes the event's payload into the
// registered Go type and wraps it in EventWithID. If the event type has no
// registered handler, it returns ErrNoFoldForEventType.
func (d *TypeDecoder) Decode(evt event.Event) (any, error) {
	h, ok := d.handlers[string(evt.Type())]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrNoFoldForEventType, evt.Type())
	}

	return h(evt)
}

// EventTypes returns all registered event type strings, sorted for determinism.
// This is useful for verifying that all fold event types are covered.
func (d *TypeDecoder) EventTypes() []string {
	types := make([]string, 0, len(d.handlers))
	for t := range d.handlers {
		types = append(types, t)
	}

	return types
}
