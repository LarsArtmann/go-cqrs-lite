package projectionadapter

import (
	"context"
	"encoding/json/v2"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
)

// PayloadDecoder converts an event type + raw payload into a typed value
// that the metaengine fold handlers expect. The consumer provides this
// because only they know the concrete types of their events.
//
// If nil, the adapter decodes the payload as map[string]any (generic JSON).
//
// PayloadDecoder is sufficient for Counter and Set queries that don't need
// the entity ID. For Map ADT queries (where the entity ID is the projection
// key), use EventDecoder via WithEventDecoder instead — it is the recommended
// decoder for all query types because it provides full event context.
type PayloadDecoder func(eventType string, payload []byte) (any, error)

// EventDecoder converts a full event into a typed value that the metaengine
// fold handlers expect. Unlike PayloadDecoder, it has access to the event's
// StreamID, metadata, and version — needed for Map ADT queries where the
// entity ID (stream ID) is the projection key.
//
// This is the RECOMMENDED decoder for all metaengine queries. Use it via
// WithEventDecoder. When set, it takes precedence over the PayloadDecoder
// passed to New.
type EventDecoder func(evt event.Event) (any, error)

// AdapterOption tunes an Adapter at construction time.
type AdapterOption func(*Adapter)

// WithEventDecoder sets an EventDecoder on the Adapter. When set, the adapter
// uses it instead of the PayloadDecoder, giving fold handlers access to the
// full event (StreamID, metadata, version). This is required for Map ADT
// queries that key on the entity ID.
//
//	adapter := projectionadapter.New("tasks", store, nil,
//		projectionadapter.WithEventDecoder(myDecoder))
func WithEventDecoder(dec EventDecoder) AdapterOption {
	return func(a *Adapter) { a.eventDecoder = dec }
}

// Adapter wraps a [metaengine.Store] as a [projection.Projection], so a
// metaengine Store can be registered with [projectionhost.Host] and process
// events through the standard projection lifecycle (checkpoint, retry, DLQ).
//
// The adapter decodes each event's payload via the configured PayloadDecoder
// (or generic JSON if none), then calls Store.Apply to route the event to
// all registered queries that listen to it.
//
// This package is intentionally separate from the core metaengine package
// to preserve metaengine's zero-dependency boundary — the adapter is the
// only place that needs event/ and projection/ imports.
type Adapter struct {
	store        *metaengine.Store
	name         string
	decoder      PayloadDecoder
	eventDecoder EventDecoder
	types        []event.Type
}

// New creates a projection.Projection backed by a metaengine Store.
// The event types are derived from the store's planned queries — every
// event type that any query listens to is included.
//
// Pass WithEventDecoder as an option when fold handlers need the full event
// (e.g. Map ADT queries keyed by stream ID).
func New(
	name string,
	store *metaengine.Store,
	decoder PayloadDecoder,
	opts ...AdapterOption,
) *Adapter {
	rawTypes := store.EventTypes()

	types := make([]event.Type, len(rawTypes))
	for i, t := range rawTypes {
		types[i] = event.Type(t)
	}

	adapter := &Adapter{
		store:   store,
		name:    name,
		decoder: decoder,
		types:   types,
	}

	for _, opt := range opts {
		opt(adapter)
	}

	return adapter
}

// Name implements projection.Projection.
func (a *Adapter) Name() string { return a.name }

// EventTypes implements projection.Projection.
func (a *Adapter) EventTypes() []event.Type { return a.types }

// Handle implements projection.Projection.
func (a *Adapter) Handle(ctx context.Context, evt event.Event) error {
	eventType := string(evt.Type())
	payload := evt.Payload()

	var decoded any

	var err error

	// EventDecoder takes precedence — it has full event context (StreamID, etc.).
	if a.eventDecoder != nil {
		decoded, err = a.eventDecoder(evt)
	} else if a.decoder != nil {
		decoded, err = a.decoder(eventType, payload)
	} else {
		err = json.Unmarshal(payload, &decoded)
	}

	if err != nil {
		return fmt.Errorf("projectionadapter: decode payload for %s: %w", eventType, err)
	}

	if err := a.store.ApplyRecord(ctx, event.AsRecord(evt), decoded); err != nil {
		return fmt.Errorf("projectionadapter: apply %s: %w", eventType, err)
	}

	return nil
}

// Compile-time assertion that Adapter implements projection.Projection.
var _ projection.Projection = (*Adapter)(nil)

// NewWithDecoder creates an Adapter using a TypeDecoder — the recommended
// constructor for all consumers. It replaces the boilerplate pattern of
// passing nil to New and overriding with WithEventDecoder:
//
// Before:
//
//	dec := buildDecoderManually() // 77-line switch/case function
//	adapter := projectionadapter.New("tasks", store, nil,
//	    projectionadapter.WithEventDecoder(dec))
//
// After:
//
//	dec := projectionadapter.NewTypeDecoder().
//	    On(evtTaskCreated, TaskCreatedPayload{}).
//	    On(evtTaskDeleted, TaskDeletedPayload{})
//	adapter := projectionadapter.NewWithDecoder("tasks", store, dec)
func NewWithDecoder(
	name string,
	store *metaengine.Store,
	decoder *TypeDecoder,
	opts ...AdapterOption,
) *Adapter {
	a := New(name, store, nil, opts...)
	a.eventDecoder = decoder.Decode

	return a
}

// RegisterWithHost is a convenience that creates an Adapter and registers it
// with a projectionhost.Host in one call. This is the one-liner for wiring
// a metaengine Store into the CQRS event sourcing lifecycle:
//
//	host, _ := projectionhost.New(journal, cpStore)
//	projectionadapter.RegisterWithHost(host, "users", store, decoder)
//	go host.Start(ctx)
//
// The event types are auto-derived from the store's planned queries.
func RegisterWithHost(
	host interface {
		Register(p projection.Projection) error
	},
	name string,
	store *metaengine.Store,
	decoder PayloadDecoder,
) error {
	adapter := New(name, store, decoder)

	if err := host.Register(adapter); err != nil {
		return fmt.Errorf("projectionadapter.RegisterWithHost: %w", err)
	}

	return nil
}
