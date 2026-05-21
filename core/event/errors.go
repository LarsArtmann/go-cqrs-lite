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

var (
	errMismatchedSlices = NewRejection(
		"event.mismatched_slices",
		"event types and payloads must have equal length",
	)
	errPayloadMarshal = NewCorruption(
		"event.payload_marshal",
		"failed to marshal event payload",
	)
	errInvalidSnapshotInterval = NewRejection(
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
	errDuplicateProjection = NewConflict("event.duplicate_projection", "duplicate projection")
	ErrStoreClosed         = NewInfrastructure("event.store_closed", "event store is closed")
	ErrBusClosed           = NewInfrastructure("event.bus_closed", "event bus is closed")
	ErrSnapshotStoreClosed = NewInfrastructure(
		"event.snapshot_store_closed",
		"snapshot store is closed",
	)
	errNilProjection      = NewInfrastructure("event.nil_projection", "nil projection")
	errNilCheckpointStore = NewInfrastructure(
		"event.nil_checkpoint_store",
		"nil checkpoint store",
	)
	ErrNilOutbox       = NewInfrastructure("event.nil_outbox", "nil outbox")
	ErrNilBus          = NewInfrastructure("event.nil_bus", "nil bus")
	ErrAlreadyStarted  = NewInfrastructure(
		"event.already_started",
		"outbox publisher already started",
	)
	ErrPublisherClosed = NewInfrastructure(
		"event.publisher_closed",
		"outbox publisher is closed",
	)
	errProjectionPanicked = NewCorruption(
		"event.projection_panicked",
		"projection handler panicked",
	)
)
