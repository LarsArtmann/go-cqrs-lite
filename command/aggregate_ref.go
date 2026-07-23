package command

import (
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// StreamType and StreamRef are type aliases for the id package types.
// These exist so command consumers can import everything from command/
// without adding a direct event/ dependency for these core identifiers.
// This is an intentional convenience re-export, not a layering violation:
// commands operate on the same aggregate identity as events.
type StreamType = id.StreamType

type StreamRef = id.StreamRef

func ParseAggregateType(s string) (StreamType, error) {
	t, err := id.ParseAggregateType(s)
	if err != nil {
		return "", errorfamily.WrapRejection(
			err,
			"command.parse_aggregate_type",
			"parse aggregate type",
		)
	}

	return t, nil
}

func NewAggregateRef(streamType StreamType, streamID id.StreamID) StreamRef {
	return id.NewAggregateRef(streamType, streamID)
}
