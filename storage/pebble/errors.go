package pebble

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrNilDatabase is returned when a store constructor is called with a nil *pebble.DB.
var ErrNilDatabase error = errorfamily.NewRejection(
	"pebble.nil_database",
	"pebble: constructor called with nil database",
)

var (
	// ErrStreamTypeMismatch is returned when an event's stream type doesn't match.
	ErrStreamTypeMismatch error = errorfamily.NewConflict(
		"pebble.stream_type_mismatch",
		"pebble: event stream type mismatch",
	)
	// ErrStreamIDMismatch is returned when an event's stream ID doesn't match.
	ErrStreamIDMismatch error = errorfamily.NewConflict(
		"pebble.stream_id_mismatch",
		"pebble: event stream ID mismatch",
	)
	// ErrVersionMismatch is returned when an event's version doesn't match.
	ErrVersionMismatch error = errorfamily.NewConflict(
		"pebble.version_mismatch",
		"pebble: event version mismatch",
	)
)

// Deprecated: use ErrStreamTypeMismatch.
var ErrAggregateTypeMismatch error = ErrStreamTypeMismatch

// Deprecated: use ErrStreamIDMismatch.
var ErrAggregateIDMismatch error = ErrStreamIDMismatch
