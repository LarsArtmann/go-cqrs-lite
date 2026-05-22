package projection

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

type Builder struct {
	name       string
	registry   *HandlerRegistry
	eventTypes []event.Type
}

func NewBuilder(name string) *Builder {
	return &Builder{
		name:     name,
		registry: NewHandlerRegistry(),
	}
}

func On[T any](b *Builder, eventType event.Type, handler func(context.Context, T) error) error {
	if handler == nil {
		return ErrNilHandler
	}

	wrapper := func(ctx context.Context, evt event.Event) error {
		var payload T

		if len(evt.Payload()) > 0 {
			err := json.Unmarshal(evt.Payload(), &payload)
			if err != nil {
				return fmt.Errorf("decode payload for %s: %w", eventType, err)
			}
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

func (b *Builder) Build() event.Projection {
	types := b.eventTypes
	if types == nil {
		types = []event.Type{}
	}

	return &builtProjection{
		name:       b.name,
		registry:   b.registry,
		eventTypes: types,
	}
}

type builtProjection struct {
	name       string
	registry   *HandlerRegistry
	eventTypes []event.Type
}

func (p *builtProjection) Name() string             { return p.name }
func (p *builtProjection) EventTypes() []event.Type { return p.eventTypes }

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
