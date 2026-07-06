package sql

import (
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// ErrNilDB is returned when a nil *sql.DB is passed to a storage constructor.
var ErrNilDB = errorfamily.NewInfrastructure(
	"storage.nil_db",
	"storage: nil database connection",
)

// ErrAggregateTypeMismatch is returned when an event's aggregate type doesn't match the expected type.
var ErrAggregateTypeMismatch = errorfamily.NewConflict(
	"storage.aggregate_type_mismatch",
	"storage: event aggregate type mismatch",
)

// ErrAggregateIDMismatch is returned when an event's aggregate ID doesn't match the expected ID.
var ErrAggregateIDMismatch = errorfamily.NewConflict(
	"storage.aggregate_id_mismatch",
	"storage: event aggregate ID mismatch",
)

// ErrVersionMismatch is returned when an event's version doesn't match the expected version.
var ErrVersionMismatch = errorfamily.NewConflict(
	"storage.version_mismatch",
	"storage: event version mismatch",
)

// ErrUnsupportedTimestamp is returned when a timestamp format cannot be parsed.
var ErrUnsupportedTimestamp = errorfamily.NewCorruption(
	"storage.unsupported_timestamp",
	"storage: unsupported timestamp format",
)

// ErrUnexpectedTimeType is returned when a time scan destination has an unexpected type.
var ErrUnexpectedTimeType = errorfamily.NewCorruption(
	"storage.unexpected_time_type",
	"storage: unexpected time type",
)

// ErrConcurrencyConflict is returned when an optimistic concurrency check fails.
var ErrConcurrencyConflict = event.ErrVersionConflict
