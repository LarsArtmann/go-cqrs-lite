package pebble

import (
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// ErrNilDatabase is returned when a store constructor is called with a nil *pebble.DB.
var ErrNilDatabase = event.NewRejection(
	"pebble.nil_database",
	"pebble: constructor called with nil database",
)

var (
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
