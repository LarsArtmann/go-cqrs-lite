package command

import (
	"context"
	"errors"
	"io"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/dispatcher/v4"
)

// Dispatcher routes commands to their handlers.
//
// The struct + NewDispatcher + Use shape is duplicated in query.Dispatcher.
// This is forced by Go's type system: command.Handler and query.Handler are
// distinct function types, so each module needs its own Dispatcher instantiation
// of the generic dispatcher.Dispatcher[H, M]. Embedding the inner dispatcher
// would eliminate the Use wrapper but expose the inner *Dispatcher field,
// breaking the encapsulation that the typed wrapper exists to provide.
// Extracting a generic TypedWrapper[H, M] in the dispatcher package would
// remove the struct duplication but at the cost of an extra public type and
// the loss of the unexported `inner` field. The current 4-statement duplication
// (struct/io.Closer/NewDispatcher/Use) is the minimum that preserves the typed,
// encapsulated public API.
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

	return dispatcher.RegisterWithWrapping(
		d.inner, string(cmdType), "command", handler,
		dispatcher.ApplyMiddleware[Handler, Middleware],
	)
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
	return d.inner.WrapCheckClosed(ErrDispatcherClosed, code, msg)
}

// Close marks the dispatcher as closed.
func (d *Dispatcher) Close() error {
	return d.inner.WrapClose("command.dispatcher_close", "close command dispatcher")
}
