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

// EventRegistration is a single event-type-to-payload-type mapping, created
// by the generic Register function. Collect them and pass to NewTypeDecoder.
type EventRegistration struct {
	eventType string
	handler   func(event.Event) (any, error)
}

// Register creates an EventRegistration that maps an event type to its
// payload Go type. The event type is an event.Type (a string typedef).
// Go infers the payload type from the sample argument.
//
// The payload is decoded via encoding/json/v2 and wrapped in EventWithID,
// giving fold handlers access to both the entity ID and the typed payload.
//
//	projectionadapter.Register(evtTaskCreated, TaskCreatedPayload{})
func Register[E any](eventType event.Type, _ E) EventRegistration {
	t := string(eventType)

	return EventRegistration{
		eventType: t,
		handler: func(evt event.Event) (any, error) {
			var p E

			if len(evt.Payload()) > 0 {
				if err := json.Unmarshal(evt.Payload(), &p); err != nil {
					return nil, fmt.Errorf("projectionadapter: decode %s: %w", t, err)
				}
			}

			return EventWithID[E]{ID: evt.StreamID().String(), Payload: p}, nil
		},
	}
}

// RegisterString is like Register but accepts a plain string event type,
// for consumers that use string constants instead of event.Type.
func RegisterString[E any](eventType string, _ E) EventRegistration {
	return EventRegistration{
		eventType: eventType,
		handler: func(evt event.Event) (any, error) {
			var p E

			if len(evt.Payload()) > 0 {
				if err := json.Unmarshal(evt.Payload(), &p); err != nil {
					return nil, fmt.Errorf("projectionadapter: decode %s: %w", eventType, err)
				}
			}

			return EventWithID[E]{ID: evt.StreamID().String(), Payload: p}, nil
		},
	}
}

// TypeDecoder eliminates the manual switch/case event decoder every consumer
// previously had to write. Register each event type with its payload Go type
// via Register, then pass the decoder to NewWithDecoder.
//
// Before (77 lines of boilerplate per consumer):
//
//	func myDecoder(evt event.Event) (any, error) {
//	    switch evt.Type() {
//	    case "task.created":
//	        var p TaskCreatedPayload
//	        json.Unmarshal(evt.Payload(), &p)
//	        return eventWithID[TaskCreatedPayload]{ID: evt.StreamID().String(), Payload: p}, nil
//	    case "task.assigned":
//	        // ... identical 5-line pattern ...
//	    // ... 9 more cases ...
//	    }
//	}
//
// After (5 lines):
//
//	dec := projectionadapter.NewTypeDecoder(
//	    projectionadapter.Register(evtTaskCreated, TaskCreatedPayload{}),
//	    projectionadapter.Register(evtTaskAssigned, TaskAssignedPayload{}),
//	    projectionadapter.Register(evtTaskDeleted, TaskDeletedPayload{}),
//	)
//	adapter := projectionadapter.NewWithDecoder("tasks", store, dec)
type TypeDecoder struct {
	handlers map[string]func(event.Event) (any, error)
}

// NewTypeDecoder creates a TypeDecoder from event registrations. Each
// registration maps an event type to its payload Go type. The decoder
// will JSON-decode each event payload, wrap it in EventWithID (with the
// stream ID as the entity key), and pass it to the metaengine fold handlers.
func NewTypeDecoder(regs ...EventRegistration) *TypeDecoder {
	d := &TypeDecoder{handlers: make(map[string]func(event.Event) (any, error), len(regs))}

	for _, r := range regs {
		d.handlers[r.eventType] = r.handler
	}

	return d
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

// EventTypes returns all registered event type strings, unsorted.
// Useful for verifying that all fold event types are covered.
func (d *TypeDecoder) EventTypes() []string {
	types := make([]string, 0, len(d.handlers))
	for t := range d.handlers {
		types = append(types, t)
	}

	return types
}
