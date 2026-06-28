// Package projection defines the Projection interface — the consumer-side
// abstraction for event-sourced read models. It is the contract implemented by
// storage.RelationalProjection, graph.GraphProjection, and stack.Materialize.
//
// The Projection interface lives here (not in event/) because projections are
// CONSUMERS of events, not producers. The event package defines what events ARE
// (Event, Store, Sink, Source, Journal, Bus); the projection package defines
// how events are CONSUMED into read models. Keeping them separate follows the
// dependency-direction principle: the projection package depends on event,
// never the reverse.
package projection

import (
	"context"
	"slices"

	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v3"
)

// Projection processes events of specific types within a projection runner.
// Implementations include storage.RelationalProjection (multi-table SQL),
// graph.GraphProjection (nodes + edges), and consumer-built projections.
type Projection interface {
	Name() string
	Handle(ctx context.Context, evt cqrsevent.Event) error
	EventTypes() []cqrsevent.Type
}

type projectionFunc struct {
	name       string
	handle     func(context.Context, cqrsevent.Event) error
	eventTypes []cqrsevent.Type
}

// NewProjection creates a Projection from a handler function and event type
// filter. Events whose type is not in eventTypes are silently skipped.
func NewProjection(
	name string,
	handle func(context.Context, cqrsevent.Event) error,
	eventTypes []cqrsevent.Type,
) Projection {
	return &projectionFunc{
		name:       name,
		handle:     handle,
		eventTypes: slices.Clone(eventTypes),
	}
}

func (p *projectionFunc) Name() string { return p.name }

func (p *projectionFunc) Handle(ctx context.Context, evt cqrsevent.Event) error {
	return p.handle(ctx, evt)
}

func (p *projectionFunc) EventTypes() []cqrsevent.Type { return slices.Clone(p.eventTypes) }

var _ Projection = (*projectionFunc)(nil)
