package watermill

import "github.com/larsartmann/go-cqrs-lite/event/v3"

// ErrMissingMetadata is returned when a required metadata field is missing from a Watermill message.
var ErrMissingMetadata = event.NewRejection(
	"watermill.missing_metadata",
	"missing required metadata",
)
