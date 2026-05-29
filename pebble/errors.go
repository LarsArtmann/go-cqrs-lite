package pebble

import "github.com/larsartmann/go-cqrs-lite/core/event"

var (
	// ErrPebbleProviderRequired is returned when no PebbleProvider is configured.
	ErrPebbleProviderRequired = event.NewInfrastructure(
		"pebble.provider_required",
		"pebble: requires a Provider: use WithPebbleProvider",
	)
	// ErrUnknownBackend is returned when an unknown event store backend is specified.
	ErrUnknownBackend = event.NewInfrastructure(
		"pebble.unknown_backend",
		"pebble: unknown event store backend",
	)
	// ErrAggregateTypeMismatch is returned when an event's aggregate type doesn't match.
	ErrAggregateTypeMismatch = event.NewConflict(
		"pebble.aggregate_type_mismatch",
		"pebble: event aggregate type mismatch",
	)
	// ErrAggregateIDMismatch is returned when an event's aggregate ID doesn't match.
	ErrAggregateIDMismatch = event.NewConflict(
		"pebble.aggregate_id_mismatch",
		"pebble: event aggregate ID mismatch",
	)
	// ErrVersionMismatch is returned when an event's version doesn't match.
	ErrVersionMismatch = event.NewConflict(
		"pebble.version_mismatch",
		"pebble: event version mismatch",
	)
)
