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
	Orchestration  = errorfamily.Orchestration
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
	ErrEmptyEventType error = errorfamily.NewRejection(
		"event.empty_event_type",
		"event type is required",
	)
	ErrNilStreamID error = errorfamily.NewRejection(
		"event.nil_stream_id",
		"stream ID is required",
	)
	ErrEmptyStreamType error = errorfamily.NewRejection(
		"event.empty_stream_type",
		"stream type is required",
	)
	ErrVersionNotPositive error = errorfamily.NewRejection(
		"event.version_not_positive",
		"version must be positive",
	)
	ErrNilPayload error = errorfamily.NewRejection(
		"event.nil_payload",
		"payload is required",
	)
	ErrMismatchedEventCount error = errorfamily.NewRejection(
		"event.mismatched_event_count",
		"event types and payloads count must match",
	)
	ErrVersionConflict error = errorfamily.NewConflict("event.version_conflict", "version conflict")
	ErrStreamNotFound  error = errorfamily.NewRejection(
		"event.stream_not_found",
		"stream not found",
	)

	// Deprecated: use ErrNilStreamID.
	ErrNilAggregateID error = ErrNilStreamID
	// Deprecated: use ErrEmptyStreamType.
	ErrEmptyAggregateType error = ErrEmptyStreamType
	// Deprecated: use ErrStreamNotFound.
	ErrAggregateNotFound error = ErrStreamNotFound
	ErrEventNotFound     error = errorfamily.NewRejection(
		"event.event_not_found",
		"event not found",
	)
	ErrStoreClosed error = errorfamily.NewInfrastructure(
		"event.store_closed",
		"event store is closed",
	)
	ErrBusClosed error = errorfamily.NewInfrastructure("event.bus_closed", "event bus is closed")
	ErrNilBus    error = errorfamily.NewInfrastructure("event.nil_bus", "nil bus")

	// Optional-capability errors returned by DecorateStore when the inner
	// store does not implement the asserted optional interface.
	ErrInnerStoreNotJournal error = errorfamily.NewRejection(
		"event.inner_store_not_journal",
		"inner store does not implement Journal",
	)
	ErrInnerStoreNotSeekable error = errorfamily.NewRejection(
		"event.inner_store_not_seekable",
		"inner store does not implement SeekableJournal",
	)
	ErrInnerStoreNotBackwards error = errorfamily.NewRejection(
		"event.inner_store_not_backwards",
		"inner store does not implement BackwardsSource",
	)
	ErrInnerStoreNotMultiSink error = errorfamily.NewRejection(
		"event.inner_store_not_multi_sink",
		"inner store does not implement MultiSink",
	)
	ErrInnerStoreNotStreaming error = errorfamily.NewRejection(
		"event.inner_store_not_streaming",
		"inner journal does not implement StreamingJournal",
	)

	// Time and date validation errors — returned by NewDate and NewWallTime.
	ErrInvalidDate error = errorfamily.NewRejection("event.invalid_date", "invalid calendar date")
	ErrInvalidHour error = errorfamily.NewRejection(
		"event.invalid_hour",
		"wall_time hour out of range [0, 23]",
	)
	ErrInvalidMinute error = errorfamily.NewRejection(
		"event.invalid_minute",
		"wall_time minute out of range [0, 59]",
	)
)
