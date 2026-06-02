package pebble

import "github.com/larsartmann/go-cqrs-lite/event/v2"

var (
	// ErrPebbleProviderRequired is returned when no PebbleProvider is configured.
	ErrPebbleProviderRequired = event.NewInfrastructure(
		"pebble.provider_required",
		"pebble: requires a Provider: use WithPebbleProvider",
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
