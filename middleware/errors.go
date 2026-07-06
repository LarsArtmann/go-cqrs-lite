package middleware

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrMeterRequired is returned by NewOTelBundle when metrics are enabled
// but a nil meter was passed. Pass WithMetricsDisabled for a tracing-only bundle.
var ErrMeterRequired = errorfamily.NewRejection(
	"middleware.meter_required",
	"meter is required when metrics are enabled (pass WithMetricsDisabled for tracing-only)",
)

// ErrValidationFailed is returned when a message fails validation.
var ErrValidationFailed = errorfamily.NewRejection(
	"middleware.validation_failed",
	"validation failed",
)

// ErrRetryExhausted is returned when all retry attempts have been exhausted.
var ErrRetryExhausted = errorfamily.NewInfrastructure(
	"middleware.retry_exhausted",
	"retry exhausted",
)

// ErrRetryCanceled is returned when a retry is canceled due to context cancellation.
var ErrRetryCanceled = errorfamily.NewInfrastructure(
	"middleware.retry_canceled",
	"retry canceled",
)

// ErrPanicRecovered is returned when a panic is recovered in a handler.
var ErrPanicRecovered = errorfamily.NewCorruption(
	"middleware.panic_recovered",
	"panic recovered",
)
