package middleware

import (
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// ErrMeterRequired is returned by NewOTelBundle when metrics are enabled
// but a nil meter was passed. Pass WithMetricsDisabled for a tracing-only bundle.
var ErrMeterRequired = event.NewRejection(
	"middleware.meter_required",
	"meter is required when metrics are enabled (pass WithMetricsDisabled for tracing-only)",
)

// ErrValidationFailed is returned when a message fails validation.
var ErrValidationFailed = event.NewRejection(
	"middleware.validation_failed",
	"validation failed",
)

// ErrRetryExhausted is returned when all retry attempts have been exhausted.
var ErrRetryExhausted = event.NewInfrastructure(
	"middleware.retry_exhausted",
	"retry exhausted",
)

// ErrRetryCanceled is returned when a retry is canceled due to context cancellation.
var ErrRetryCanceled = event.NewInfrastructure(
	"middleware.retry_canceled",
	"retry canceled",
)

// ErrPanicRecovered is returned when a panic is recovered in a handler.
var ErrPanicRecovered = event.NewCorruption(
	"middleware.panic_recovered",
	"panic recovered",
)
