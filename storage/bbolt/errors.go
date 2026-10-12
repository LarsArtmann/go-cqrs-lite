package bbolt

import errorfamily "github.com/larsartmann/go-error-family"

var (
	ErrNilDatabase error = errorfamily.NewRejection(
		"bbolt.nil_database", "database must not be nil")

	ErrVersionMismatch error = errorfamily.NewConflict(
		"bbolt.version_mismatch", "event version does not match expected sequence")

	ErrStreamTypeMismatch error = errorfamily.NewConflict(
		"bbolt.stream_type_mismatch", "event stream type does not match ref")

	ErrStreamIDMismatch error = errorfamily.NewConflict(
		"bbolt.stream_id_mismatch", "event stream ID does not match ref")
)
