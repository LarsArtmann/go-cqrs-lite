package event

import "context"

// Projection processes events to build a read model or view.
// Implementations must be idempotent — the same event may be
// delivered more than once during recovery or retry scenarios.
type Projection interface {
	// Name returns the unique identifier for this projection.
	// Used for checkpoint tracking and logging.
	Name() string

	// Handle processes a single event.
	// Implementations should use event.DecodePayload[T] for
	// type-safe payload deserialization.
	Handle(ctx context.Context, evt Event) error

	// EventTypes returns the set of event types this projection
	// subscribes to. Return nil to subscribe to all events.
	EventTypes() []Type
}

// ProjectionFunc is a convenience type for creating projections
// from a single handler function.
type ProjectionFunc struct {
	name       string
	handle     func(ctx context.Context, evt Event) error
	eventTypes []Type
}

// NewProjection creates a ProjectionFunc with the given name and handler.
func NewProjection(
	name string,
	handle func(ctx context.Context, evt Event) error,
	eventTypes []Type,
) *ProjectionFunc {
	return &ProjectionFunc{
		name:       name,
		handle:     handle,
		eventTypes: eventTypes,
	}
}

// Name returns the projection name.
func (p *ProjectionFunc) Name() string { return p.name }

// Handle delegates to the handler function.
func (p *ProjectionFunc) Handle(ctx context.Context, evt Event) error {
	return p.handle(ctx, evt)
}

// EventTypes returns the subscribed event types.
func (p *ProjectionFunc) EventTypes() []Type { return p.eventTypes }

var _ Projection = (*ProjectionFunc)(nil)
