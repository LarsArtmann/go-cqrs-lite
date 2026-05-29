package event

import "context"

// Projection processes events of specific types within a projection runner.
type Projection interface {
	Name() string
	Handle(ctx context.Context, evt Event) error
	EventTypes() []Type
}

// BatchProjection is an optional interface that projections can implement
// to process multiple events in a single call for higher throughput.
// When implemented, the runner will call HandleBatch instead of Handle
// for each event batch.
type BatchProjection interface {
	Projection
	HandleBatch(ctx context.Context, events []Event) error
}

type projectionFunc struct {
	name       string
	handle     func(ctx context.Context, evt Event) error
	eventTypes []Type
}

// NewProjection creates a Projection from a handler function and event type filter.
func NewProjection(
	name string,
	handle func(ctx context.Context, evt Event) error,
	eventTypes []Type,
) Projection {
	return &projectionFunc{
		name:       name,
		handle:     handle,
		eventTypes: eventTypes,
	}
}

func (p *projectionFunc) Name() string { return p.name }

func (p *projectionFunc) Handle(ctx context.Context, evt Event) error {
	return p.handle(ctx, evt)
}

func (p *projectionFunc) EventTypes() []Type { return p.eventTypes }

var _ Projection = (*projectionFunc)(nil)
