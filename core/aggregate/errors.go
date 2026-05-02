package aggregate

import "github.com/cockroachdb/errors"

var (
	// ErrNilAggregateID is returned when a nil aggregate ID is passed to NewCore.
	ErrNilAggregateID = errors.New("aggregate ID is required")

	// ErrEmptyAggregateType is returned when an empty aggregate type is passed to NewCore.
	ErrEmptyAggregateType = errors.New("aggregate type is required")

	// ErrNilStore is returned when a nil event store is passed to NewRepository.
	ErrNilStore = errors.New("aggregate: nil store")

	// ErrNilBus is returned when a nil event bus is passed to NewRepository.
	ErrNilBus = errors.New("aggregate: nil bus")
)
