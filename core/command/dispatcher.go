// Package command provides command dispatching for CQRS.
package command

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/dispatcher"
)

// Dispatcher routes commands to their handlers.
type Dispatcher struct {
	dispatcher.CatalogDispatcher[Type, dispatcher.CatalogEntry]

	inner *dispatcher.Dispatcher[Handler, Middleware]
}

var _ io.Closer = (*Dispatcher)(nil)

// NewDispatcher creates a new command dispatcher.
func NewDispatcher() *Dispatcher {
	d := &Dispatcher{} //nolint:exhaustruct // embedded generic fields require Init method
	d.InitCatalogDispatcher()
	d.inner = dispatcher.NewDispatcher[Handler, Middleware]()

	return d
}

// Use adds middleware to the dispatcher.
func (d *Dispatcher) Use(middleware ...Middleware) {
	d.inner.Use(middleware...)
}

// Register binds a handler to a command type.
func (d *Dispatcher) Register(cmdType Type, handler Handler) error {
	err := d.inner.Lifecycle.CheckClosed(ErrDispatcherClosed)
	if err != nil {
		return fmt.Errorf("registering command type %s: %w", cmdType, err)
	}

	err = d.inner.Register(
		string(cmdType),
		handler,
		func(m Middleware, h Handler) Handler {
			return m(h)
		},
	)
	if err != nil {
		return fmt.Errorf("registering handler for command type %s: %w", cmdType, err)
	}

	return nil
}

// Dispatch sends a command to its registered handler.
func (d *Dispatcher) Dispatch(ctx context.Context, cmd Command) error {
	wrapped, err := d.inner.Dispatch(string(cmd.Type()))
	if err != nil {
		if errors.Is(err, dispatcher.ErrHandlerNotFound) {
			return fmt.Errorf("%w: command type: %s", ErrHandlerNotFound, cmd.Type())
		}

		return fmt.Errorf("command type %s: %w", cmd.Type(), err)
	}

	return wrapped(ctx, cmd)
}

// Close marks the dispatcher as closed.
func (d *Dispatcher) Close() error {
	return d.inner.Close() //nolint:wrapcheck // lifecycle Close is self-descriptive
}
