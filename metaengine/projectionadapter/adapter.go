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
type PayloadDecoder func(eventType string, payload []byte) (any, error)

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
	store   *metaengine.Store
	name    string
	decoder PayloadDecoder
	types   []event.Type
}

// New creates a projection.Projection backed by a metaengine Store.
// The event types are derived from the store's planned queries — every
// event type that any query listens to is included.
func New(
	name string,
	store *metaengine.Store,
	decoder PayloadDecoder,
) *Adapter {
	rawTypes := store.EventTypes()

	types := make([]event.Type, len(rawTypes))
	for i, t := range rawTypes {
		types[i] = event.Type(t)
	}

	return &Adapter{
		store:   store,
		name:    name,
		decoder: decoder,
		types:   types,
	}
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

	if a.decoder != nil {
		decoded, err = a.decoder(eventType, payload)
	} else {
		err = json.Unmarshal(payload, &decoded)
	}

	if err != nil {
		return fmt.Errorf("projectionadapter: decode payload for %s: %w", eventType, err)
	}

	if err := a.store.Apply(ctx, eventType, decoded); err != nil {
		return fmt.Errorf("projectionadapter: apply %s: %w", eventType, err)
	}

	return nil
}

// Compile-time assertion that Adapter implements projection.Projection.
var _ projection.Projection = (*Adapter)(nil)

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
		Register(projection.Projection) error
	},
	name string,
	store *metaengine.Store,
	decoder PayloadDecoder,
) error {
	adapter := New(name, store, decoder)
	return host.Register(adapter)
}
