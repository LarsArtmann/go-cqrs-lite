package id

import (
	"fmt"
	"time"
)

// Deprecated: use StreamMarker. AggregateMarker is retained as a type alias
// for backward compatibility with consumer code that embeds it for branding.
type AggregateMarker = StreamMarker

// Deprecated: use StreamID. AggregateID is retained as a type alias.
type AggregateID = StreamID

// Deprecated: use NewStreamID.
func NewAggregateID() StreamID { return NewStreamID() }

// Deprecated: use ParseStreamID.
func ParseAggregateID(s string) (StreamID, error) { return ParseStreamID(s) }

// Deprecated: use ParseStreamIDStrict.
func ParseAggregateIDStrict(s string) (StreamID, error) { return ParseStreamIDStrict(s) }

// Deprecated: use IsStreamIDULID.
func IsAggregateIDULID(id StreamID) bool { return IsStreamIDULID(id) }

// Deprecated: use StreamTimestamp.
func AggregateTimestamp(id StreamID) (time.Time, error) { return StreamTimestamp(id) }

// Deprecated: use DeriveStreamID.
func DeriveAggregateID(namespace string, keys ...string) StreamID {
	return DeriveStreamID(namespace, keys...)
}

// Deprecated: use StreamIDFrom.
func AggregateIDFrom(s fmt.Stringer) StreamID { return StreamIDFrom(s) }
