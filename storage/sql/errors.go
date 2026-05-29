package sql

import "github.com/larsartmann/go-cqrs-lite/event"

// ErrNilDB is returned when a nil *sql.DB is passed to a storage constructor.
var ErrNilDB = event.NewInfrastructure(
	"storage.nil_db",
	"storage: nil database connection",
)

// ErrAggregateTypeMismatch is returned when an event's aggregate type doesn't match the expected type.
var ErrAggregateTypeMismatch = event.NewConflict(
	"storage.aggregate_type_mismatch",
	"storage: event aggregate type mismatch",
)

// ErrAggregateIDMismatch is returned when an event's aggregate ID doesn't match the expected ID.
var ErrAggregateIDMismatch = event.NewConflict(
	"storage.aggregate_id_mismatch",
	"storage: event aggregate ID mismatch",
)

// ErrVersionMismatch is returned when an event's version doesn't match the expected version.
var ErrVersionMismatch = event.NewConflict(
	"storage.version_mismatch",
	"storage: event version mismatch",
)

// ErrUnsupportedTimestamp is returned when a timestamp format cannot be parsed.
var ErrUnsupportedTimestamp = event.NewCorruption(
	"storage.unsupported_timestamp",
	"storage: unsupported timestamp format",
)

// ErrUnexpectedTimeType is returned when a time scan destination has an unexpected type.
var ErrUnexpectedTimeType = event.NewCorruption(
	"storage.unexpected_time_type",
	"storage: unexpected time type",
)

// ErrConcurrencyConflict is returned when an optimistic concurrency check fails.
var ErrConcurrencyConflict = event.ErrVersionConflict
