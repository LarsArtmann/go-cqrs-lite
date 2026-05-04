package middleware

import "errors"

// ErrValidationFailed is returned when a message fails validation.
var ErrValidationFailed = errors.New("validation failed")

// ErrRetryExhausted is returned when all retry attempts have been exhausted.
var ErrRetryExhausted = errors.New("retry exhausted")

// ErrRetryCanceled is returned when a retry is canceled due to context cancellation.
var ErrRetryCanceled = errors.New("retry canceled")

// ErrPanicRecovered is returned when a panic is recovered in a handler.
var ErrPanicRecovered = errors.New("panic recovered")
