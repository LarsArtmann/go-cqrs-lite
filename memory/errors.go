package memory

import (
	"errors"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// ErrHandlerNil is returned when a nil handler is passed to Subscribe or SubscribeAll.
var ErrHandlerNil = errors.New("handler must not be nil")

func init() { //nolint:gochecknoinits
	event.RegisterClassification(ErrHandlerNil, event.Rejection)
}
