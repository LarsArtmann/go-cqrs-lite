package event

import (
	"context"
	"slices"
)

// Projection processes events of specific types within a projection runner.
type Projection interface {
	Name() string
	Handle(ctx context.Context, evt Event) error
	EventTypes() []Type
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
		eventTypes: slices.Clone(eventTypes),
	}
}

func (p *projectionFunc) Name() string { return p.name }

func (p *projectionFunc) Handle(ctx context.Context, evt Event) error {
	return p.handle(ctx, evt)
}

func (p *projectionFunc) EventTypes() []Type { return slices.Clone(p.eventTypes) }

var _ Projection = (*projectionFunc)(nil)
