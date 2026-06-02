package projection

import "github.com/larsartmann/go-cqrs-lite/event/v2"

var (
	// ErrNilHandler is returned when a nil projection is registered.
	ErrNilHandler = event.NewRejection(
		"projection.nil_handler",
		"projection: nil handler",
	)

	// ErrNilBus is returned when a nil event subscriber is passed to NewRunner.
	ErrNilBus = event.NewInfrastructure(
		"projection.nil_bus",
		"projection: nil bus",
	)

	// ErrNilCheckpoint is returned when a nil checkpoint store is passed to NewRunner.
	ErrNilCheckpoint = event.NewInfrastructure(
		"projection.nil_checkpoint",
		"projection: nil checkpoint store",
	)

	// ErrNoProjections is returned when Run is called without any registered projections.
	ErrNoProjections = event.NewRejection(
		"projection.no_projections",
		"projection: no projections registered",
	)

	// ErrDuplicateProjection is returned when a projection with the same name is registered twice.
	ErrDuplicateProjection = event.NewConflict(
		"projection.duplicate_projection",
		"projection: duplicate projection name",
	)
)
