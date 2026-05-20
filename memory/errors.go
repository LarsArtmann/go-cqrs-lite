package memory

import "github.com/larsartmann/go-cqrs-lite/core/event"

// ErrHandlerNil is returned when a nil handler is passed to Subscribe or SubscribeAll.
var ErrHandlerNil = event.NewRejection(
	"memory.handler_nil",
	"handler must not be nil",
)
