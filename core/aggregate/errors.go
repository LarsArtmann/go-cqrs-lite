package aggregate

import (
	"errors"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

var (
	// ErrNilAggregateID is returned when a nil aggregate ID is passed to NewCore.
	ErrNilAggregateID = errors.New("aggregate ID is required")

	// ErrEmptyAggregateType is returned when an empty aggregate type is passed to NewCore.
	ErrEmptyAggregateType = errors.New("aggregate type is required")

	// ErrNilStore is returned when a nil event store is passed to NewRepository.
	ErrNilStore = errors.New("aggregate: nil store")

	// ErrNilBus is returned when a nil event publisher is passed to NewRepository.
	ErrNilBus = errors.New("aggregate: nil bus")
)

func init() { //nolint:gochecknoinits
	event.RegisterClassification(ErrNilAggregateID, event.Rejection)
	event.RegisterClassification(ErrEmptyAggregateType, event.Rejection)
	event.RegisterClassification(ErrNilStore, event.Infrastructure)
	event.RegisterClassification(ErrNilBus, event.Infrastructure)
}
