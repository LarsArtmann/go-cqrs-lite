package id

import (
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"
)

// StreamType is a type identifier for event streams.
type StreamType string

// String returns the stream type as a string.
func (s StreamType) String() string { return string(s) }

// IsZero reports whether the stream type is empty.
func (s StreamType) IsZero() bool { return s == "" }

// ErrEmptyStreamType is returned when a stream type is required but empty.
var ErrEmptyStreamType = errorfamily.NewRejection(
	"id.empty_stream_type",
	"stream type is required",
)

// ParseStreamType validates and returns a StreamType.
// Returns an error if empty.
func ParseStreamType(s string) (StreamType, error) {
	if s == "" {
		return "", ErrEmptyStreamType
	}

	return StreamType(s), nil
}

// StreamRef uniquely identifies an event stream by its type and ID.
// Use this to pass stream identity as a single value instead of separate params.
type StreamRef struct {
	Type StreamType
	ID   StreamID
}

func (r StreamRef) String() string {
	return r.Type.String() + ":" + r.ID.String()
}

// StreamKey returns the canonical key for an event stream.
func (r StreamRef) StreamKey() string {
	return r.String()
}

// NewStreamRef creates a StreamRef from a type and ID.
func NewStreamRef(streamType StreamType, streamID StreamID) StreamRef {
	return StreamRef{Type: streamType, ID: streamID}
}

// IsZero returns true if both Type and ID are their zero values.
func (r StreamRef) IsZero() bool {
	return r.Type == "" && r.ID.IsZero()
}

// Validate returns an error if Type is empty or ID is zero.
func (r StreamRef) Validate() error {
	if r.Type == "" {
		return ErrEmptyStreamType
	}

	if r.ID.IsZero() {
		return errorfamily.NewRejection("id.nil_stream_id", "stream ID is required")
	}

	return nil
}

// Verify StreamRef satisfies fmt.Stringer at compile time.
//
//nolint:exhaustruct // zero-value proves interface is satisfied at compile time
var _ fmt.Stringer = StreamRef{}
