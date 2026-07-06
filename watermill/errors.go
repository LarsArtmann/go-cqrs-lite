package watermill

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrMissingMetadata is returned when a required metadata field is missing from a Watermill message.
var ErrMissingMetadata = errorfamily.NewRejection(
	"watermill.missing_metadata",
	"missing required metadata",
)
