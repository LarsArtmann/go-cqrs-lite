package pebble

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrNilDatabase is returned when a store constructor is called with a nil *pebble.DB.
var ErrNilDatabase = errorfamily.NewRejection(
	"pebble.nil_database",
	"pebble: constructor called with nil database",
)

var (
	// ErrAggregateTypeMismatch is returned when an event's aggregate type doesn't match.
	ErrAggregateTypeMismatch = errorfamily.NewConflict(
		"pebble.aggregate_type_mismatch",
		"pebble: event aggregate type mismatch",
	)
	// ErrAggregateIDMismatch is returned when an event's aggregate ID doesn't match.
	ErrAggregateIDMismatch = errorfamily.NewConflict(
		"pebble.aggregate_id_mismatch",
		"pebble: event aggregate ID mismatch",
	)
	// ErrVersionMismatch is returned when an event's version doesn't match.
	ErrVersionMismatch = errorfamily.NewConflict(
		"pebble.version_mismatch",
		"pebble: event version mismatch",
	)
)
