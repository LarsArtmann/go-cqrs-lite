package event

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// Family classifies an error into one of five categories:
// Rejection, Conflict, Transient, Corruption, or Infrastructure.
type (
	Family = errorfamily.Family

	// Error is a structured error with family, code, message, and optional cause.
	Error = errorfamily.Error
)

// Error family constants for classification.
const (
	// Rejection indicates the request was invalid or unauthorized.
	Rejection = errorfamily.Rejection
	// Conflict indicates a state conflict (e.g., version mismatch).
	Conflict = errorfamily.Conflict
	// Transient indicates a temporary failure that may succeed on retry.
	Transient = errorfamily.Transient
	// Corruption indicates data integrity violation.
	Corruption = errorfamily.Corruption
	// Infrastructure indicates a system-level failure (e.g., network, storage).
	Infrastructure = errorfamily.Infrastructure
)

// Classify returns the error family for the given error.
// Unknown errors default to Transient.
func Classify(err error) Family { return errorfamily.Classify(err) }

// IsRetryable returns true for Transient errors.
func IsRetryable(err error) bool { return errorfamily.IsRetryable(err) }

// NewRejection creates a Rejection-classified error with code and message.
func NewRejection(code, msg string) *Error {
	return errorfamily.NewRejection(code, msg)
}

// NewConflict creates a Conflict-classified error with code and message.
func NewConflict(code, msg string) *Error { return errorfamily.NewConflict(code, msg) }

// NewTransient creates a Transient-classified error with code and message.
func NewTransient(code, msg string) *Error {
	return errorfamily.NewTransient(code, msg)
}

// NewCorruption creates a Corruption-classified error with code and message.
func NewCorruption(code, msg string) *Error {
	return errorfamily.NewCorruption(code, msg)
}

// NewInfrastructure creates an Infrastructure-classified error with code and message.
func NewInfrastructure(code, msg string) *Error {
	return errorfamily.NewInfrastructure(code, msg)
}

// Wrap creates a classified error wrapping an existing error.
func Wrap(err error, family Family, code, msg string) *Error {
	return errorfamily.Wrap(err, family, code, msg)
}

// WrapRejection wraps err as a Rejection.
func WrapRejection(err error, code, msg string) *Error {
	return errorfamily.WrapRejection(err, code, msg)
}

// WrapConflict wraps err as a Conflict.
func WrapConflict(err error, code, msg string) *Error {
	return errorfamily.WrapConflict(err, code, msg)
}

// WrapTransient wraps err as a Transient.
func WrapTransient(err error, code, msg string) *Error {
	return errorfamily.WrapTransient(err, code, msg)
}

// WrapCorruption wraps err as Corruption.
func WrapCorruption(err error, code, msg string) *Error {
	return errorfamily.WrapCorruption(err, code, msg)
}

// WrapInfrastructure wraps err as Infrastructure.
func WrapInfrastructure(err error, code, msg string) *Error {
	return errorfamily.WrapInfrastructure(err, code, msg)
}

// Wrapf wraps err with formatted message.
func Wrapf(err error, family Family, code, format string, args ...any) *Error {
	return errorfamily.Wrapf(err, family, code, format, args...)
}

// Newf creates a classified error with formatted message.
func Newf(family Family, code, format string, args ...any) *Error {
	return errorfamily.Newf(family, code, format, args...)
}

// ExitCode returns a process exit code derived from the error family.
func ExitCode(err error) int { return errorfamily.ExitCode(err) }

var (
	ErrEmptyEventType     = NewRejection("event.empty_event_type", "event type is required")
	ErrNilAggregateID     = NewRejection("event.nil_aggregate_id", "aggregate ID is required")
	ErrEmptyAggregateType = NewRejection(
		"event.empty_aggregate_type",
		"aggregate type is required",
	)
	ErrVersionNotPositive = NewRejection(
		"event.version_not_positive",
		"version must be positive",
	)
	ErrNilPayload           = NewRejection("event.nil_payload", "payload is required")
	ErrMismatchedEventCount = NewRejection(
		"event.mismatched_event_count",
		"event types and payloads count must match",
	)
	ErrVersionConflict   = NewConflict("event.version_conflict", "version conflict")
	ErrAggregateNotFound = NewRejection("event.aggregate_not_found", "aggregate not found")
	ErrBinaryNotFound   = NewRejection("event.binary_not_found", "binary data not found in event metadata")
	ErrStoreClosed       = NewInfrastructure("event.store_closed", "event store is closed")
	ErrBusClosed         = NewInfrastructure("event.bus_closed", "event bus is closed")
	ErrNilBus            = NewInfrastructure("event.nil_bus", "nil bus")
)
