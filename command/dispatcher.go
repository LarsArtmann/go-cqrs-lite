// Package command provides command dispatching for CQRS.
package command

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/go-cqrs-lite/internal/dispatcher"
)

// Dispatcher routes commands to their handlers.
type Dispatcher struct {
	handlers   map[Type]Handler
	lifecycle  dispatcher.Lifecycle
	middleware dispatcher.MiddlewareChain[Handler, Middleware]
}

// NewDispatcher creates a new command dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: make(map[Type]Handler),
	}
}

// Use adds middleware to the dispatcher.
func (d *Dispatcher) Use(middleware ...Middleware) {
	d.middleware.Add(middleware...)
}

// Register binds a handler to a command type.
func (d *Dispatcher) Register(cmdType Type, handler Handler) error {
	if err := d.lifecycle.CheckClosed(ErrDispatcherClosed); err != nil {
		return err
	}
	d.handlers[cmdType] = handler
	return nil
}

// Dispatch sends a command to its registered handler.
func (d *Dispatcher) Dispatch(ctx context.Context, cmd Command) error {
	if err := d.lifecycle.CheckClosed(ErrDispatcherClosed); err != nil {
		return err
	}

	handler, ok := d.handlers[cmd.Type()]
	if !ok {
		return errors.Wrapf(ErrHandlerNotFound, "command type: %s", cmd.Type())
	}

	wrapped := d.middleware.Apply(handler, func(m Middleware, h Handler) Handler {
		return m(h)
	})

	return wrapped(ctx, cmd)
}

// Close marks the dispatcher as closed.
func (d *Dispatcher) Close() error {
	return d.lifecycle.Close()
}
