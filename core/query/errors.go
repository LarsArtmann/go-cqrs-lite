package query

import (
	"errors"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// ErrQueryNotSupported is returned when a query type is not supported.
var ErrQueryNotSupported = errors.New("query not supported")

// ErrDispatcherClosed is returned when the dispatcher is closed.
var ErrDispatcherClosed = errors.New("query dispatcher is closed")

// ErrEmptyQueryType is returned when a query is created with an empty type.
var ErrEmptyQueryType = errors.New("query type is required (got empty)")

func init() { //nolint:gochecknoinits // registers error classifications for cross-package Classify()
	event.RegisterClassification(ErrQueryNotSupported, event.Rejection)
	event.RegisterClassification(ErrDispatcherClosed, event.Infrastructure)
	event.RegisterClassification(ErrEmptyQueryType, event.Rejection)
}
