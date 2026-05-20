package aggregate

import "github.com/larsartmann/go-cqrs-lite/core/event"

var (
	// ErrNilAggregateID is returned when a nil aggregate ID is passed to NewCore.
	ErrNilAggregateID = event.NewRejection(
		"aggregate.nil_aggregate_id",
		"aggregate ID is required",
	)

	// ErrEmptyAggregateType is returned when an empty aggregate type is passed to NewCore.
	ErrEmptyAggregateType = event.NewRejection(
		"aggregate.empty_aggregate_type",
		"aggregate type is required",
	)

	// ErrNilStore is returned when a nil event store is passed to NewRepository.
	ErrNilStore = event.NewInfrastructure(
		"aggregate.nil_store",
		"aggregate: nil store",
	)

	// ErrNilBus is returned when a nil event publisher is passed to NewRepository.
	ErrNilBus = event.NewInfrastructure(
		"aggregate.nil_bus",
		"aggregate: nil bus",
	)
)
