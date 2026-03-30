package event

import "github.com/cockroachdb/errors"

// ErrEventNotFound is returned when an event is not found.
var ErrEventNotFound = errors.New("event not found")

// ErrVersionConflict is returned when there is a version conflict.
var ErrVersionConflict = errors.New("version conflict")

// ErrAggregateNotFound is returned when an aggregate is not found.
var ErrAggregateNotFound = errors.New("aggregate not found")

// ErrInvalidEventType is returned when an event type is invalid.
var ErrInvalidEventType = errors.New("invalid event type")

// ErrStoreClosed is returned when the event store is closed.
var ErrStoreClosed = errors.New("event store is closed")

// ErrBusClosed is returned when the event bus is closed.
var ErrBusClosed = errors.New("event bus is closed")
