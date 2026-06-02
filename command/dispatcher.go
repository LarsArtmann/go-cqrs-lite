package command

import (
	"context"
	"errors"
	"io"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/dispatcher/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// Dispatcher routes commands to their handlers.
type Dispatcher struct {
	inner *dispatcher.Dispatcher[Handler, Middleware]
}

var _ io.Closer = (*Dispatcher)(nil)

// NewDispatcher creates a new command dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		inner: dispatcher.NewDispatcher[Handler, Middleware](),
	}
}

// Use adds middleware to the dispatcher.
func (d *Dispatcher) Use(middleware ...Middleware) {
	d.inner.Use(middleware...)
}

// Register binds a handler to a command type.
func (d *Dispatcher) Register(cmdType Type, handler Handler) error {
	err := d.checkClosed("command.register_failed", "registering command type "+string(cmdType))
	if err != nil {
		return err
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
	err := d.checkClosed("command.dispatch_failed", "dispatching command type "+string(cmd.Type()))
	if err != nil {
		return err
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

func (d *Dispatcher) checkClosed(code, msg string) error {
	err := d.inner.Lifecycle.CheckClosed(ErrDispatcherClosed)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, code, msg)
	}

	return nil
}

// Close marks the dispatcher as closed.
func (d *Dispatcher) Close() error {
	err := d.inner.Close()
	if err != nil {
		return event.WrapInfrastructure(err, "command.dispatcher_close",
			"close command dispatcher")
	}

	return nil
}
