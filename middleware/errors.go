package middleware

import (
	"errors"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// ErrValidationFailed is returned when a message fails validation.
var ErrValidationFailed = errors.New("validation failed")

// ErrRetryExhausted is returned when all retry attempts have been exhausted.
var ErrRetryExhausted = errors.New("retry exhausted")

// ErrRetryCanceled is returned when a retry is canceled due to context cancellation.
var ErrRetryCanceled = errors.New("retry canceled")

// ErrPanicRecovered is returned when a panic is recovered in a handler.
var ErrPanicRecovered = errors.New("panic recovered")

func init() { //nolint:gochecknoinits
	event.RegisterClassification(ErrValidationFailed, event.Rejection)
	event.RegisterClassification(ErrRetryExhausted, event.Infrastructure)
	event.RegisterClassification(ErrRetryCanceled, event.Infrastructure)
	event.RegisterClassification(ErrPanicRecovered, event.Corruption)
}
