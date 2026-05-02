package projection

import "github.com/cockroachdb/errors"

var (
	// ErrNilHandler is returned when a nil projection is registered.
	ErrNilHandler = errors.New("projection: nil handler")

	// ErrNilBus is returned when a nil event bus is passed to NewRunner.
	ErrNilBus = errors.New("projection: nil bus")

	// ErrNilCheckpoint is returned when a nil checkpoint store is passed to NewRunner.
	ErrNilCheckpoint = errors.New("projection: nil checkpoint store")

	// ErrNoProjections is returned when Run is called without any registered projections.
	ErrNoProjections = errors.New("projection: no projections registered")
)
