// Package command provides command dispatching for CQRS.
package command

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/dispatcher"
)

// Dispatcher routes commands to their handlers.
type Dispatcher struct {
	dispatcher.CatalogDispatcher[Type, CatalogMeta]

	base dispatcher.BaseDispatcher[Handler, Middleware]
}

// NewDispatcher creates a new command dispatcher.
func NewDispatcher() *Dispatcher {
	d := &Dispatcher{} //nolint:exhaustruct // embedded generic fields require Init method
	d.InitCatalogDispatcher()
	d.base = dispatcher.NewBaseDispatcher[Handler, Middleware]()

	return d
}

// Use adds middleware to the dispatcher.
func (d *Dispatcher) Use(middleware ...Middleware) {
	d.base.Use(middleware...)
}

// Register binds a handler to a command type.
func (d *Dispatcher) Register(cmdType Type, handler Handler) error {
	err := d.base.Lifecycle().CheckClosed(ErrDispatcherClosed)
	if err != nil {
		return errors.Wrapf(err, "registering command type %s", cmdType)
	}

	err = d.base.Register(string(cmdType), handler)
	if err != nil {
		return errors.Wrapf(err, "register handler for command type %s", cmdType)
	}

	return nil
}

// Dispatch sends a command to its registered handler.
func (d *Dispatcher) Dispatch(ctx context.Context, cmd Command) error {
	err := d.base.Lifecycle().CheckClosed(ErrDispatcherClosed)
	if err != nil {
		return errors.Wrapf(err, "dispatching command %s", cmd.Type())
	}

	_, ok := d.base.GetHandler(string(cmd.Type()))
	if !ok {
		return errors.Wrapf(ErrHandlerNotFound, "command type: %s", cmd.Type())
	}

	wrapped, err := d.base.Dispatch(
		string(cmd.Type()),
		func(m Middleware, h Handler) Handler {
			return m(h)
		},
	)
	if err != nil {
		return errors.Wrapf(err, "command type: %s", cmd.Type())
	}

	return wrapped(ctx, cmd)
}

// Close marks the dispatcher as closed.
func (d *Dispatcher) Close() error {
	err := d.base.Lifecycle().Close()
	if err != nil {
		return errors.Wrapf(err, "close command dispatcher")
	}

	return nil
}
