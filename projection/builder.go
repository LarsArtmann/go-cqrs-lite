package projection

import (
	"context"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// Builder constructs a Projection with type-safe event handlers.
// Use On[T] to register handlers, then call Build to create the Projection.
type Builder struct {
	name       string
	registry   *HandlerRegistry
	eventTypes []event.Type
}

// NewBuilder creates a Builder for a projection with the given name.
func NewBuilder(name string) *Builder {
	return &Builder{
		name:     name,
		registry: NewHandlerRegistry(),
	}
}

// On registers a type-safe handler for the given event type.
// The payload is decoded using the provided codec, which must match the event's encoding.
// Returns ErrNilHandler if handler is nil.
func On[T any](
	b *Builder,
	eventType event.Type,
	c codec.Codec,
	handler func(context.Context, T) error,
) error {
	if handler == nil {
		return ErrNilHandler
	}

	wrapper := func(ctx context.Context, evt event.Event) error {
		payload, err := event.DecodePayload[T](evt, c)
		if err != nil {
			return err
		}

		return handler(ctx, payload)
	}

	err := b.registry.On(eventType, wrapper)
	if err != nil {
		return err
	}

	b.eventTypes = append(b.eventTypes, eventType)

	return nil
}

// Build creates an event.Projection from the registered handlers.
func (b *Builder) Build() event.Projection {
	types := b.eventTypes
	if types == nil {
		types = []event.Type{}
	}

	return &builtProjection{
		name:       b.name,
		registry:   b.registry,
		eventTypes: slices.Clone(types),
	}
}

type builtProjection struct {
	name       string
	registry   *HandlerRegistry
	eventTypes []event.Type
}

func (p *builtProjection) Name() string             { return p.name }
func (p *builtProjection) EventTypes() []event.Type { return slices.Clone(p.eventTypes) }

func (p *builtProjection) Handle(ctx context.Context, evt event.Event) error {
	handlers := p.registry.Lookup(evt.Type())

	for _, h := range handlers {
		err := h(ctx, evt)
		if err != nil {
			return err
		}
	}

	return nil
}
