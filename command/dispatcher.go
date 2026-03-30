// Package command provides command dispatching for CQRS.
package command

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/go-cqrs-lite/internal/dispatcher"
)

// Dispatcher routes commands to their handlers.
type Dispatcher struct {
	inner *dispatcher.Dispatcher[Handler, Middleware]
}

// NewDispatcher creates a new command dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		inner: dispatcher.NewDispatcher[Handler, Middleware](),
	}
}

// Use adds middleware to the dispatcher.
func (d *Dispatcher) Use(middleware ...Middleware) {
	// Access the Middleware field directly through the inner dispatcher
	d.inner.Middleware.Add(middleware...)
}

// Register binds a handler to a command type.
func (d *Dispatcher) Register(cmdType Type, handler Handler) error {
	if err := d.inner.Lifecycle.CheckClosed(ErrDispatcherClosed); err != nil {
		return errors.Wrapf(err, "registering command type %s", cmdType)
	}
	return d.inner.Register(string(cmdType), handler)
}

// Dispatch sends a command to its registered handler.
func (d *Dispatcher) Dispatch(ctx context.Context, cmd Command) error {
	if err := d.inner.Lifecycle.CheckClosed(ErrDispatcherClosed); err != nil {
		return errors.Wrapf(err, "dispatching command %s", cmd.Type())
	}

	handler, ok := d.inner.Handlers[string(cmd.Type())]
	if !ok {
		return errors.Wrapf(ErrHandlerNotFound, "command type: %s", cmd.Type())
	}

	wrapped := d.inner.Middleware.Apply(handler, func(m Middleware, h Handler) Handler {
		return m(h)
	})

	return wrapped(ctx, cmd)
}

// Close marks the dispatcher as closed.
func (d *Dispatcher) Close() error {
	return d.inner.Lifecycle.Close()
}
