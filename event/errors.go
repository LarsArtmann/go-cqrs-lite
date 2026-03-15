package event

import "github.com/cockroachdb/errors"

var (
	ErrEventNotFound     = errors.New("event not found")
	ErrVersionConflict   = errors.New("version conflict")
	ErrAggregateNotFound = errors.New("aggregate not found")
	ErrInvalidEventType  = errors.New("invalid event type")
	ErrStoreClosed       = errors.New("event store is closed")
	ErrBusClosed         = errors.New("event bus is closed")
)
