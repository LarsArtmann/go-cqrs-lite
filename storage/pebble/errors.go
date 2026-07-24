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
	// ErrStreamTypeMismatch is returned when an event's stream type doesn't match.
	ErrStreamTypeMismatch = errorfamily.NewConflict(
		"pebble.aggregate_type_mismatch",
		"pebble: event aggregate type mismatch",
	)
	// ErrStreamIDMismatch is returned when an event's stream ID doesn't match.
	ErrStreamIDMismatch = errorfamily.NewConflict(
		"pebble.aggregate_id_mismatch",
		"pebble: event aggregate ID mismatch",
	)
	// ErrVersionMismatch is returned when an event's version doesn't match.
	ErrVersionMismatch = errorfamily.NewConflict(
		"pebble.version_mismatch",
		"pebble: event version mismatch",
	)
)

// Deprecated: use ErrStreamTypeMismatch.
var ErrAggregateTypeMismatch = ErrStreamTypeMismatch

// Deprecated: use ErrStreamIDMismatch.
var ErrAggregateIDMismatch = ErrStreamIDMismatch
