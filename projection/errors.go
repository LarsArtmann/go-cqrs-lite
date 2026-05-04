package projection

import (
	"errors"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

var (
	// ErrNilHandler is returned when a nil projection is registered.
	ErrNilHandler = errors.New("projection: nil handler")

	// ErrNilBus is returned when a nil event subscriber is passed to NewRunner.
	ErrNilBus = errors.New("projection: nil bus")

	// ErrNilCheckpoint is returned when a nil checkpoint store is passed to NewRunner.
	ErrNilCheckpoint = errors.New("projection: nil checkpoint store")

	// ErrNoProjections is returned when Run is called without any registered projections.
	ErrNoProjections = errors.New("projection: no projections registered")
)

func init() { //nolint:gochecknoinits
	event.RegisterClassification(ErrNilHandler, event.Rejection)
	event.RegisterClassification(ErrNilBus, event.Infrastructure)
	event.RegisterClassification(ErrNilCheckpoint, event.Infrastructure)
	event.RegisterClassification(ErrNoProjections, event.Rejection)
}
