package command

import (
	"context"
	"fmt"
	"slices"
	"sync"

	errorfamily "github.com/larsartmann/go-error-family"
)

// MemoryBus is an in-memory implementation of [Bus]. It dispatches commands
// synchronously to registered handlers. It is safe for concurrent use.
//
// Use [NewMemoryBus] to create one. Commands published via [MemoryBus.Publish]
// are delivered to all handlers registered for the command type, plus all
// catch-all handlers registered via [MemoryBus.SubscribeAll].
//
// Middleware registered via [MemoryBus.Use] wraps every handler in the chain.
type MemoryBus struct {
	mu          sync.RWMutex
	handlers    map[Type][]Handler
	allHandlers []Handler
	middleware  []Middleware
}

// NewMemoryBus creates a new in-memory command bus.
func NewMemoryBus() *MemoryBus {
	return &MemoryBus{ //nolint:exhaustruct // handlers lazily initialized
		handlers: make(map[Type][]Handler),
	}
}

var (
	errNilBusHandler      = errorfamily.NewRejection("command.nil_handler", "command: nil handler")
	errNilBusSubscribeAll = errorfamily.NewRejection(
		"command.nil_subscribe_all",
		"command: subscribe-all: nil handler",
	)
)

// Subscribe registers a handler for a specific command type.
// Multiple handlers may be registered for the same type; all are called
// synchronously on Publish. Returns an error if handler is nil.
func (b *MemoryBus) Subscribe(cmdType Type, handler Handler) error {
	if handler == nil {
		return errorfamily.WrapRejection(errNilBusHandler, "command.memory_bus.subscribe",
			fmt.Sprintf("subscribe %s", cmdType))
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[cmdType] = append(b.handlers[cmdType], handler)

	return nil
}

// SubscribeAll registers a catch-all handler invoked for every published
// command, regardless of type. Useful for audit logging.
func (b *MemoryBus) SubscribeAll(handler Handler) error {
	if handler == nil {
		return errNilBusSubscribeAll
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.allHandlers = append(b.allHandlers, handler)

	return nil
}

// Use appends middleware to the handler chain. Middleware applies to all
// handlers, wrapping them in registration order. Must be called before
// Publish; calling Use after Publish does not affect in-flight commands.
func (b *MemoryBus) Use(mw ...Middleware) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.middleware = append(b.middleware, mw...)

	return nil
}

// Publish dispatches commands to all registered handlers. Each command is
// delivered to handlers registered for its type, then to all catch-all
// handlers. All handlers for a single command run synchronously; an error
// from any handler stops further dispatch for that command.
func (b *MemoryBus) Publish(ctx context.Context, cmds ...Command) error {
	b.mu.RLock()
	mw := make([]Middleware, len(b.middleware))
	copy(mw, b.middleware)
	b.mu.RUnlock()

	for _, cmd := range cmds {
		err := b.dispatch(ctx, cmd, mw)
		if err != nil {
			return errorfamily.Wrap(err, errorfamily.Classify(err),
				"command.memory_bus.publish",
				fmt.Sprintf("publish %s", cmd.Type()))
		}
	}

	return nil
}

func (b *MemoryBus) dispatch(ctx context.Context, cmd Command, mw []Middleware) error {
	b.mu.RLock()
	typed := b.handlers[cmd.Type()]
	all := b.allHandlers
	b.mu.RUnlock()

	// Wrap each handler with middleware.
	applyMW := func(h Handler) Handler {
		for _, v := range slices.Backward(mw) {
			h = v(h)
		}

		return h
	}

	// Dispatch to typed handlers first.
	for _, h := range typed {
		err := applyMW(h)(ctx, cmd)
		if err != nil {
			return err
		}
	}

	// Then to catch-all handlers.
	for _, h := range all {
		err := applyMW(h)(ctx, cmd)
		if err != nil {
			return err
		}
	}

	return nil
}

// Compile-time assertion.
var _ Bus = (*MemoryBus)(nil)
