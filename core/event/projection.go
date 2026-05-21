package event

import "context"

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
