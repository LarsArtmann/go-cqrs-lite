package id

// Deprecated: use StreamType. AggregateType is retained as a type alias.
type AggregateType = StreamType

// Deprecated: use ErrEmptyStreamType.
var ErrEmptyAggregateType = ErrEmptyStreamType

// Deprecated: use ParseStreamType.
func ParseAggregateType(s string) (StreamType, error) { return ParseStreamType(s) }

// Deprecated: use StreamRef. AggregateRef is retained as a type alias.
type AggregateRef = StreamRef

// Deprecated: use NewStreamRef.
func NewAggregateRef(streamType StreamType, streamID StreamID) StreamRef {
	return NewStreamRef(streamType, streamID)
}
