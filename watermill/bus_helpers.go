package watermill

import (
	"sync"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// errBusClosed wraps event.ErrBusClosed for the given Watermill component and
// domain noun. Shared by CommandBus and EventBus methods so the error
// wrapping pattern stays consistent.
func errBusClosed(component, noun string) error {
	return errorfamily.WrapInfrastructure(event.ErrBusClosed,
		component, noun+" bus is closed")
}

// registerTypedHandler is the shared subscribe logic for both CommandBus and
// EventBus. It locks the bus, checks the closed guard, appends the handler to
// the type-keyed map, rebuilds the handler chain, and ensures the background
// subscription is running.
func registerTypedHandler[K ~string, H any](
	mu *sync.Mutex,
	closed bool,
	handlers map[K][]H,
	key K,
	handler H,
	component, noun string,
	rebuild func(),
	ensure func(),
) error {
	mu.Lock()
	defer mu.Unlock()

	if closed {
		return errBusClosed(component, noun)
	}

	handlers[key] = append(handlers[key], handler)
	rebuild()
	ensure()

	return nil
}

// registerAllHandler is the shared SubscribeAll logic for both CommandBus and
// EventBus. It takes a pointer to the allHandlers slice so the append updates
// the struct field.
func registerAllHandler[H any](
	mu *sync.Mutex,
	closed bool,
	handlers *[]H,
	handler H,
	component, noun string,
	rebuild func(),
	ensure func(),
) error {
	mu.Lock()
	defer mu.Unlock()

	if closed {
		return errBusClosed(component, noun)
	}

	*handlers = append(*handlers, handler)
	rebuild()
	ensure()

	return nil
}
