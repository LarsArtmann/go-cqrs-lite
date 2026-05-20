package storage

import (
	"errors"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

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
var ErrNilDB = errors.New("storage: nil database connection")

// ErrAggregateTypeMismatch is returned when an event's aggregate type doesn't match the expected type.
var ErrAggregateTypeMismatch = errors.New("storage: event aggregate type mismatch")

// ErrAggregateIDMismatch is returned when an event's aggregate ID doesn't match the expected ID.
var ErrAggregateIDMismatch = errors.New("storage: event aggregate ID mismatch")

// ErrVersionMismatch is returned when an event's version doesn't match the expected version.
var ErrVersionMismatch = errors.New("storage: event version mismatch")

// ErrPebbleProviderRequired is returned when no PebbleProvider is configured.
var ErrPebbleProviderRequired = errors.New(
	"storage: pebble requires a Provider: use WithPebbleProvider",
)

// ErrUnknownBackend is returned when an unknown event store backend is specified.
var ErrUnknownBackend = errors.New("storage: unknown event store backend")

// ErrUnsupportedTimestamp is returned when a timestamp format cannot be parsed.
var ErrUnsupportedTimestamp = errors.New("storage: unsupported timestamp format")

// ErrTursoMemorySync is returned when trying to sync an in-memory Turso database.
var ErrTursoMemorySync = errors.New("storage: turso sync requires a file path for dbPath")

// ErrUnexpectedTimeType is returned when a time scan destination has an unexpected type.
var ErrUnexpectedTimeType = errors.New("storage: unexpected time type")

func init() { //nolint:gochecknoinits
	event.RegisterClassification(ErrNilDB, event.Infrastructure)
	event.RegisterClassification(ErrAggregateTypeMismatch, event.Conflict)
	event.RegisterClassification(ErrAggregateIDMismatch, event.Conflict)
	event.RegisterClassification(ErrVersionMismatch, event.Conflict)
	event.RegisterClassification(ErrPebbleProviderRequired, event.Infrastructure)
	event.RegisterClassification(ErrUnknownBackend, event.Infrastructure)
	event.RegisterClassification(ErrUnsupportedTimestamp, event.Corruption)
	event.RegisterClassification(ErrTursoMemorySync, event.Rejection)
	event.RegisterClassification(ErrUnexpectedTimeType, event.Corruption)
}
