package command

import (
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// StreamType and StreamRef are type aliases for the id package types.
// These exist so command consumers can import everything from command/
// without adding a direct event/ dependency for these core identifiers.
// This is an intentional convenience re-export, not a layering violation:
// commands operate on the same stream identity as events.
type StreamType = id.StreamType

type StreamRef = id.StreamRef

// Deprecated: use StreamType.
type AggregateType = StreamType

// Deprecated: use StreamRef.
type AggregateRef = StreamRef

func ParseStreamType(s string) (StreamType, error) {
	t, err := id.ParseStreamType(s)
	if err != nil {
		return "", errorfamily.WrapRejection(
			err,
			"command.parse_stream_type",
			"parse stream type",
		)
	}

	return t, nil
}

func NewStreamRef(streamType StreamType, streamID id.StreamID) StreamRef {
	return id.NewStreamRef(streamType, streamID)
}

// Deprecated: use ParseStreamType.
func ParseAggregateType(s string) (StreamType, error) { return ParseStreamType(s) }

// Deprecated: use NewStreamRef.
func NewAggregateRef(streamType StreamType, streamID id.StreamID) StreamRef {
	return NewStreamRef(streamType, streamID)
}
