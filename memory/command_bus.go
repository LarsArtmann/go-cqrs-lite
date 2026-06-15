package memory

import (
	"context"
	"fmt"
	"io"
	"slices"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/dispatcher/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// MemoryCommandBus is an in-memory implementation of command.Bus.
// It mirrors memory.MemoryBus for the command side — enabling pub/sub
// semantics for commands with middleware support.
type MemoryCommandBus struct {
	dispatcher.Lifecycle

	mu          sync.RWMutex
	handlers    map[command.Type][]command.Handler
	allHandlers []command.Handler
	middleware  []command.Middleware
	cached      command.Handler
}

var (
	_ command.Bus = (*MemoryCommandBus)(nil)
	_ io.Closer   = (*MemoryCommandBus)(nil)
)

func NewMemoryCommandBus() *MemoryCommandBus {
	b := &MemoryCommandBus{ //nolint:exhaustruct // embedded Lifecycle
		handlers: make(map[command.Type][]command.Handler),
	}
	b.rebuildChain()

	return b
}

func (b *MemoryCommandBus) rebuildChain() {
	chain := command.Handler(func(ctx context.Context, cmd command.Command) error {
		return b.notify(ctx, cmd)
	})

	for _, mw := range slices.Backward(b.middleware) {
		chain = mw(chain)
	}

	b.cached = chain
}

func (b *MemoryCommandBus) notify(ctx context.Context, cmd command.Command) error {
	cmdType := cmd.Type()

	b.mu.RLock()
	typeHandlers, hasType := b.handlers[cmdType]
	allHandlers := b.allHandlers
	b.mu.RUnlock()

	if !hasType && len(allHandlers) == 0 {
		return nil
	}

	for _, handler := range typeHandlers {
		err := handler(ctx, cmd)
		if err != nil {
			return event.WrapInfrastructure(err, "memory.command_bus", fmt.Sprintf("handler for %s", cmdType))
		}
	}

	for _, handler := range allHandlers {
		err := handler(ctx, cmd)
		if err != nil {
			return event.WrapInfrastructure(err, "memory.command_bus", fmt.Sprintf("all-handler for %s", cmdType))
		}
	}

	return nil
}

func (b *MemoryCommandBus) Publish(ctx context.Context, cmds ...command.Command) error {
	err := b.CheckClosed(command.ErrDispatcherClosed)
	if err != nil {
		return event.WrapInfrastructure(err, "memory.command_bus_closed", "command bus is closed")
	}

	b.mu.RLock()
	cached := b.cached
	b.mu.RUnlock()

	for _, cmd := range cmds {
		err := cached(ctx, cmd)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *MemoryCommandBus) Subscribe(cmdType command.Type, handler command.Handler) error {
	err := b.CheckClosed(command.ErrDispatcherClosed)
	if err != nil {
		return event.WrapInfrastructure(err, "memory.command_bus_closed", "command bus is closed")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[cmdType] = append(b.handlers[cmdType], handler)

	return nil
}

func (b *MemoryCommandBus) SubscribeAll(handler command.Handler) error {
	err := b.CheckClosed(command.ErrDispatcherClosed)
	if err != nil {
		return event.WrapInfrastructure(err, "memory.command_bus_closed", "command bus is closed")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.allHandlers = append(b.allHandlers, handler)

	return nil
}

func (b *MemoryCommandBus) Use(middleware ...command.Middleware) error {
	err := b.CheckClosed(command.ErrDispatcherClosed)
	if err != nil {
		return event.WrapInfrastructure(err, "memory.command_bus_closed", "command bus is closed")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.middleware = append(b.middleware, middleware...)
	b.rebuildChain()

	return nil
}

func (b *MemoryCommandBus) Close() error {
	return b.Lifecycle.Close() //nolint:wrapcheck // transparent delegation
}
