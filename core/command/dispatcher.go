// Package command provides command dispatching for CQRS.
package command

import (
	"context"
	"errors"
	"io"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/dispatcher"
)

// Dispatcher routes commands to their handlers.
type Dispatcher struct {
	dispatcher.CatalogDispatcher[Type, dispatcher.HandlerMeta]

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
		return errorfamily.WrapInfrastructure(
			err,
			"command.register_failed",
			"registering command type "+string(cmdType),
		)
	}

	err = d.inner.Register(
		string(cmdType),
		handler,
		func(m Middleware, h Handler) Handler {
			return m(h)
		},
	)
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"command.register_handler_failed",
			"registering handler for command type "+string(cmdType),
		)
	}

	return nil
}

// Dispatch sends a command to its registered handler.
func (d *Dispatcher) Dispatch(ctx context.Context, cmd Command) error {
	err := d.inner.Lifecycle.CheckClosed(ErrDispatcherClosed)
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"command.dispatch_failed",
			"dispatching command type "+string(cmd.Type()),
		)
	}

	wrapped, err := d.inner.Dispatch(string(cmd.Type()))
	if err != nil {
		if errors.Is(err, dispatcher.ErrHandlerNotFound) {
			return errorfamily.WrapRejection(
				ErrHandlerNotFound,
				"command.handler_not_found",
				"handler not found for command type "+string(cmd.Type()),
			)
		}

		return errorfamily.Wrap(
			err,
			errorfamily.Classify(err),
			"command.handler_failed",
			"command type "+string(cmd.Type()),
		)
	}

	return wrapped(ctx, cmd)
}

// Close marks the dispatcher as closed.
func (d *Dispatcher) Close() error {
	return d.inner.Close() //nolint:wrapcheck // lifecycle Close is self-descriptive
}
