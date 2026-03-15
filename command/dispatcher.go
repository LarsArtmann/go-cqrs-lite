package command

import (
	"context"
	"sync"

	"github.com/cockroachdb/errors"
)

// Dispatcher routes commands to their handlers
type Dispatcher struct {
	mu         sync.RWMutex
	handlers   map[Type]Handler
	middleware []Middleware
	closed     bool
}

// NewDispatcher creates a new command dispatcher
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: make(map[Type]Handler),
	}
}

// Use adds middleware to the dispatcher
func (d *Dispatcher) Use(middleware ...Middleware) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.middleware = append(d.middleware, middleware...)
}

// Register binds a handler to a command type
func (d *Dispatcher) Register(commandType Type, handler Handler) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return ErrDispatcherClosed
	}

	d.handlers[commandType] = handler
	return nil
}

// Dispatch sends a command to its registered handler
func (d *Dispatcher) Dispatch(ctx context.Context, cmd Command) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return ErrDispatcherClosed
	}

	handler, ok := d.handlers[cmd.Type()]
	if !ok {
		return errors.Wrapf(ErrHandlerNotFound, "command type: %s", cmd.Type())
	}

	wrapped := handler
	for i := len(d.middleware) - 1; i >= 0; i-- {
		wrapped = d.middleware[i](wrapped)
	}

	return wrapped(ctx, cmd)
}

// Close marks the dispatcher as closed
func (d *Dispatcher) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.closed = true
	return nil
}
