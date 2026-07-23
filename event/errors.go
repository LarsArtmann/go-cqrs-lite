package event

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// Backward-compatible type aliases. These are the SAME types as in go-error-family;
// the aliases exist so existing consumer code that references event.Family or
// event.Error continues to compile. New code should import go-error-family
// directly for error taxonomy, construction, and classification.
type (
	Family = errorfamily.Family
	Error  = errorfamily.Error
)

// Backward-compatible family constants (same values as go-error-family).
const (
	Rejection      = errorfamily.Rejection
	Conflict       = errorfamily.Conflict
	Transient      = errorfamily.Transient
	Corruption     = errorfamily.Corruption
	Infrastructure = errorfamily.Infrastructure
)

// Event-domain sentinel errors. These are the only error values the event
// package owns. For error construction, classification, wrapping, and retry
// checks, import go-error-family directly:
//
//	import errorfamily "github.com/larsartmann/go-error-family"
//
//	 classified := errorfamily.Classify(err)
//	 retryable  := errorfamily.IsRetryable(err)
//	 wrapped    := errorfamily.WrapRejection(err, "my.code", "message")
var (
	ErrEmptyEventType = errorfamily.NewRejection(
		"event.empty_event_type",
		"event type is required",
	)
	ErrNilAggregateID = errorfamily.NewRejection(
		"event.nil_aggregate_id",
		"aggregate ID is required",
	)
	ErrEmptyAggregateType = errorfamily.NewRejection(
		"event.empty_aggregate_type",
		"aggregate type is required",
	)
	ErrVersionNotPositive = errorfamily.NewRejection(
		"event.version_not_positive",
		"version must be positive",
	)
	ErrNilPayload           = errorfamily.NewRejection("event.nil_payload", "payload is required")
	ErrMismatchedEventCount = errorfamily.NewRejection(
		"event.mismatched_event_count",
		"event types and payloads count must match",
	)
	ErrVersionConflict   = errorfamily.NewConflict("event.version_conflict", "version conflict")
	ErrAggregateNotFound = errorfamily.NewRejection(
		"event.aggregate_not_found",
		"aggregate not found",
	)
	ErrEventNotFound  = errorfamily.NewRejection("event.event_not_found", "event not found")
	ErrBinaryNotFound = errorfamily.NewRejection(
		"event.binary_not_found",
		"binary data not found in event metadata",
	)
	ErrStoreClosed = errorfamily.NewInfrastructure("event.store_closed", "event store is closed")
	ErrBusClosed   = errorfamily.NewInfrastructure("event.bus_closed", "event bus is closed")
	ErrNilBus      = errorfamily.NewInfrastructure("event.nil_bus", "nil bus")

	// Time and date validation errors — returned by NewDate and NewWallTime.
	ErrInvalidDate = errorfamily.NewRejection("event.invalid_date", "invalid calendar date")
	ErrInvalidHour = errorfamily.NewRejection(
		"event.invalid_hour",
		"wall_time hour out of range [0, 23]",
	)
	ErrInvalidMinute = errorfamily.NewRejection(
		"event.invalid_minute",
		"wall_time minute out of range [0, 59]",
	)
)
