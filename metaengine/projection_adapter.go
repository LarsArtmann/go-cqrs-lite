package metaengine

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
)

// PayloadDecoder converts an event type + raw payload into a typed value
// that the metaengine fold handlers expect. The consumer provides this
// because only they know the concrete types of their events.
//
// If nil, the adapter decodes the payload as map[string]any (generic JSON).
type PayloadDecoder func(eventType string, payload []byte) (any, error)

// ProjectionAdapter wraps a [Store] as a [projection.Projection], so a
// metaengine Store can be registered with [projectionhost.Host] and process
// events through the standard projection lifecycle (checkpoint, retry, DLQ).
//
// The adapter decodes each event's payload via the configured PayloadDecoder
// (or generic JSON if none), then calls Store.Apply to route the event to
// all registered queries that listen to it.
type ProjectionAdapter struct {
	store   *Store
	name    string
	decoder PayloadDecoder
	types   []event.Type
}

// NewProjectionAdapter creates a projection.Projection backed by a metaengine Store.
// The eventTypes are derived from the store's planned queries — every event
// type that any query listens to is included.
func NewProjectionAdapter(
	name string,
	store *Store,
	decoder PayloadDecoder,
) *ProjectionAdapter {
	return &ProjectionAdapter{
		store:   store,
		name:    name,
		decoder: decoder,
		types:   collectEventTypes(store),
	}
}

func collectEventTypes(store *Store) []event.Type {
	if store == nil || store.plan == nil {
		return nil
	}

	seen := make(map[string]bool)

	for _, q := range store.queries {
		for et := range q.foldByEvent {
			seen[et] = true
		}
	}

	result := make([]event.Type, 0, len(seen))
	for et := range seen {
		result = append(result, event.Type(et))
	}

	return result
}

func (a *ProjectionAdapter) Name() string { return a.name }

func (a *ProjectionAdapter) EventTypes() []event.Type { return a.types }

func (a *ProjectionAdapter) Handle(ctx context.Context, evt event.Event) error {
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
		return fmt.Errorf("metaengine: decode payload for %s: %w", eventType, err)
	}

	return a.store.Apply(ctx, eventType, decoded)
}

// Compile-time assertion that ProjectionAdapter implements projection.Projection.
var _ projection.Projection = (*ProjectionAdapter)(nil)
