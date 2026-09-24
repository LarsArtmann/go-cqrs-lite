package sql

import (
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// ErrNilDB is returned when a nil *sql.DB is passed to a storage constructor.
var ErrNilDB error = errorfamily.NewInfrastructure(
	"storage.nil_db",
	"storage: nil database connection",
)

// ErrClosed is returned when a store operation is attempted after Close.
var ErrClosed error = errorfamily.NewInfrastructure(
	"storage.closed",
	"storage: store is closed",
)

// ErrStreamTypeMismatch is returned when an event's stream type doesn't match the expected type.
var ErrStreamTypeMismatch error = errorfamily.NewConflict(
	"storage.stream_type_mismatch",
	"storage: event stream type mismatch",
)

// ErrAggregateTypeMismatch is retained as a deprecated alias for ErrStreamTypeMismatch.
//
// Deprecated: use ErrStreamTypeMismatch.
var ErrAggregateTypeMismatch error = ErrStreamTypeMismatch

// ErrStreamIDMismatch is returned when an event's stream ID doesn't match the expected ID.
var ErrStreamIDMismatch error = errorfamily.NewConflict(
	"storage.stream_id_mismatch",
	"storage: event stream ID mismatch",
)

// ErrAggregateIDMismatch is retained as a deprecated alias for ErrStreamIDMismatch.
//
// Deprecated: use ErrStreamIDMismatch.
var ErrAggregateIDMismatch error = ErrStreamIDMismatch

// ErrVersionMismatch is returned when an event's version doesn't match the expected version.
var ErrVersionMismatch error = errorfamily.NewConflict(
	"storage.version_mismatch",
	"storage: event version mismatch",
)

// ErrUnsupportedTimestamp is returned when a timestamp format cannot be parsed.
var ErrUnsupportedTimestamp error = errorfamily.NewCorruption(
	"storage.unsupported_timestamp",
	"storage: unsupported timestamp format",
)

// ErrUnexpectedTimeType is returned when a time scan destination has an unexpected type.
var ErrUnexpectedTimeType error = errorfamily.NewCorruption(
	"storage.unexpected_time_type",
	"storage: unexpected time type",
)

// ErrConcurrencyConflict is returned when an optimistic concurrency check fails.
var ErrConcurrencyConflict error = event.ErrVersionConflict
