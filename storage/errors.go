package storage

import "github.com/larsartmann/go-cqrs-lite/core/event"

// OutboxStatus represents the status of an outbox entry.
type OutboxStatus string

const (
	// OutboxStatusPending indicates the entry has not yet been published.
	OutboxStatusPending OutboxStatus = "pending"
	// OutboxStatusAcked indicates the entry has been successfully published.
	OutboxStatusAcked OutboxStatus = "acked"
)

// String returns the underlying string value.
func (s OutboxStatus) String() string { return string(s) }

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

// ErrPebbleProviderRequired is returned when no PebbleProvider is configured.
var ErrPebbleProviderRequired = event.NewInfrastructure(
	"storage.pebble_provider_required",
	"storage: pebble requires a Provider: use WithPebbleProvider",
)

// ErrUnknownBackend is returned when an unknown event store backend is specified.
var ErrUnknownBackend = event.NewInfrastructure(
	"storage.unknown_backend",
	"storage: unknown event store backend",
)

// ErrUnsupportedTimestamp is returned when a timestamp format cannot be parsed.
var ErrUnsupportedTimestamp = event.NewCorruption(
	"storage.unsupported_timestamp",
	"storage: unsupported timestamp format",
)

// ErrTursoMemorySync is returned when trying to sync an in-memory Turso database.
var ErrTursoMemorySync = event.NewRejection(
	"storage.turso_memory_sync",
	"storage: turso sync requires a file path for dbPath",
)

// ErrUnexpectedTimeType is returned when a time scan destination has an unexpected type.
var ErrUnexpectedTimeType = event.NewCorruption(
	"storage.unexpected_time_type",
	"storage: unexpected time type",
)
