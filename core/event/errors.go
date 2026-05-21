package event

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

type (
	Family = errorfamily.Family
	Error  = errorfamily.Error
)

const (
	Rejection      = errorfamily.Rejection
	Conflict       = errorfamily.Conflict
	Transient      = errorfamily.Transient
	Corruption     = errorfamily.Corruption
	Infrastructure = errorfamily.Infrastructure
)

func Classify(err error) Family  { return errorfamily.Classify(err) }
func IsRetryable(err error) bool { return errorfamily.IsRetryable(err) }
func RegisterClassification(sentinel error, family Family) {
	errorfamily.RegisterClassification(sentinel, family)
}

func NewRejection(code, msg string) *Error {
	return errorfamily.NewRejection(code, msg)
}

func NewConflict(code, msg string) *Error { return errorfamily.NewConflict(code, msg) }

func NewTransient(code, msg string) *Error {
	return errorfamily.NewTransient(code, msg)
}

func NewCorruption(code, msg string) *Error {
	return errorfamily.NewCorruption(code, msg)
}

func NewInfrastructure(code, msg string) *Error {
	return errorfamily.NewInfrastructure(code, msg)
}

// Wrap wraps an existing error with a family, code, and message.
func Wrap(err error, family Family, code, msg string) *Error {
	return errorfamily.Wrap(err, family, code, msg)
}

// WrapRejection wraps an error as a Rejection.
func WrapRejection(err error, code, msg string) *Error {
	return errorfamily.WrapRejection(err, code, msg)
}

// WrapConflict wraps an error as a Conflict.
func WrapConflict(err error, code, msg string) *Error {
	return errorfamily.WrapConflict(err, code, msg)
}

// WrapTransient wraps an error as a Transient (retryable).
func WrapTransient(err error, code, msg string) *Error {
	return errorfamily.WrapTransient(err, code, msg)
}

// WrapCorruption wraps an error as a Corruption.
func WrapCorruption(err error, code, msg string) *Error {
	return errorfamily.WrapCorruption(err, code, msg)
}

// WrapInfrastructure wraps an error as an Infrastructure error.
func WrapInfrastructure(err error, code, msg string) *Error {
	return errorfamily.WrapInfrastructure(err, code, msg)
}

// WrapFrom wraps an error while preserving its classified family.
// Use this when you don't know the error's family but want to add context.
func WrapFrom(err error, code, msg string) *Error {
	return errorfamily.Wrap(err, Classify(err), code, msg)
}

var (
	ErrMismatchedSlices = NewRejection(
		"event.mismatched_slices",
		"event types and payloads must have equal length",
	)
	ErrPayloadMarshal = NewCorruption(
		"event.payload_marshal",
		"failed to marshal event payload",
	)
	ErrInvalidSnapshotInterval = NewRejection(
		"event.invalid_snapshot_interval",
		"snapshot interval must be positive",
	)
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
	ErrVersionConflict     = NewConflict("event.version_conflict", "version conflict")
	ErrAggregateNotFound   = NewRejection("event.aggregate_not_found", "aggregate not found")
	ErrSnapshotNotFound    = NewRejection("event.snapshot_not_found", "snapshot not found")
	ErrDuplicateProjection = NewConflict("event.duplicate_projection", "duplicate projection")
	ErrStoreClosed         = NewInfrastructure("event.store_closed", "event store is closed")
	ErrBusClosed           = NewInfrastructure("event.bus_closed", "event bus is closed")
	ErrSnapshotStoreClosed = NewInfrastructure(
		"event.snapshot_store_closed",
		"snapshot store is closed",
	)
	ErrNilProjection      = NewInfrastructure("event.nil_projection", "nil projection")
	ErrNilCheckpointStore = NewInfrastructure(
		"event.nil_checkpoint_store",
		"nil checkpoint store",
	)
	ErrNilOutbox      = NewInfrastructure("event.nil_outbox", "nil outbox")
	ErrNilBus         = NewInfrastructure("event.nil_bus", "nil bus")
	ErrAlreadyStarted = NewInfrastructure(
		"event.already_started",
		"outbox publisher already started",
	)
	ErrPublisherClosed = NewInfrastructure(
		"event.publisher_closed",
		"outbox publisher is closed",
	)
	ErrProjectionPanicked = NewCorruption(
		"event.projection_panicked",
		"projection handler panicked",
	)
)
