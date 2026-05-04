package decider

import (
	"errors"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// ErrNilStore is returned by NewRepository when the event store is nil.
var ErrNilStore = errors.New("event store is required")

// ErrNilBus is returned by NewRepository when the event publisher is nil.
var ErrNilBus = errors.New("event bus is required")

// ErrNilFold is returned by NewRepository when the decider Fold function is nil.
var ErrNilFold = errors.New("fold function is required")

// ErrLoadFailed is returned when loading events from the store fails.
var ErrLoadFailed = errors.New("failed to load events")

// ErrFoldFailed is returned when folding an event onto state fails.
var ErrFoldFailed = errors.New("failed to fold events")

// ErrSaveFailed is returned when saving events to the store fails.
var ErrSaveFailed = errors.New("failed to save events")

func init() { //nolint:gochecknoinits // registers error classifications for cross-package Classify()
	event.RegisterClassification(ErrNilStore, event.Infrastructure)
	event.RegisterClassification(ErrNilBus, event.Infrastructure)
	event.RegisterClassification(ErrNilFold, event.Rejection)
	event.RegisterClassification(ErrLoadFailed, event.Transient)
	event.RegisterClassification(ErrFoldFailed, event.Corruption)
	event.RegisterClassification(ErrSaveFailed, event.Transient)
}
