package id

import (
	"fmt"
	"strconv"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

// StreamType is a type identifier for event streams.
// art-dupl:accept ADR-0152 v5 copy-forward twin of the v4 train; removed with v4 in T26
type StreamType string

// String returns the stream type as a string.
func (s StreamType) String() string { return string(s) }

// IsZero reports whether the stream type is empty.
func (s StreamType) IsZero() bool { return s == "" }

// ErrEmptyStreamType is returned when a stream type is required but empty.
var ErrEmptyStreamType error = errorfamily.NewRejection(
	"id.empty_stream_type",
	"stream type is required",
)

// ParseStreamType validates and returns a StreamType.
// Returns an error if empty.
func ParseStreamType(s string) (StreamType, error) {
	if s == "" {
		return "", ErrEmptyStreamType
	}

	// art-dupl:accept ADR-0152 v5 copy-forward twin of the v4 train (region start moved by the v4 UTF-8 boundary fix); removed with v4 in T26
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

// StreamKey returns the canonical STORAGE key for an event stream:
// "<type>:<bare id>" — the bare wire form, never the branded display form
// (mirrors v4's 2026-10-17 disk-truth contract).
func (r StreamRef) StreamKey() string {
	return r.Type.String() + ":" + r.ID.Get()
}

// NewStreamRef creates a StreamRef from a type and ID, enforcing the
// ADR-0166 validity contract at construction: an empty type and a zero ID
// are both rejected (owner ruling 2026-08-22, Appendix B). Adapters that
// cannot surface an error use [NewStreamRefOrZero]; deliberately partial
// refs (tests, wire validation) use the struct literal — the constructor is
// the validating door, the literal the trusted one.
func NewStreamRef(streamType StreamType, streamID StreamID) (StreamRef, error) {
	ref := StreamRef{Type: streamType, ID: streamID}
	if err := ref.Validate(); err != nil {
		return StreamRef{}, err
	}

	return ref, nil
}

// NewStreamRefOrZero creates a StreamRef, returning the zero StreamRef when
// the parts would not validate. It is the producer-side counterpart to
// [NewStreamRef] for adapters that cannot return an error: a Record either
// carries a well-formed StreamRef or none at all — never a malformed ref
// that fails Validate far from its cause.
func NewStreamRefOrZero(streamType StreamType, streamID StreamID) StreamRef {
	ref, err := NewStreamRef(streamType, streamID)
	if err != nil {
		return StreamRef{} //nolint:exhaustruct_v5 // zero value on the error path
	}

	return ref
}

// ParseStreamRef parses the canonical storage key form "type:bare-id"
// (the inverse of [StreamRef.StreamKey]) back into a StreamRef. The key
// splits on the FIRST colon: the type owns everything before it, the ID
// everything after (StreamIDs accept any non-empty string, so IDs may
// themselves contain colons). Both parts are required: a missing
// separator, an empty type, or an empty ID returns a Rejection.
func ParseStreamRef(key string) (StreamRef, error) {
	typePart, idPart, found := strings.Cut(key, ":")
	if !found {
		return StreamRef{}, errorfamily.NewRejection(
			"id.parse_stream_ref",
			"stream key must be \"type:id\", got no separator in "+strconv.Quote(key),
		)
	}

	streamType, err := ParseStreamType(typePart)
	if err != nil {
		return StreamRef{}, err
	}

	streamID, err := ParseStreamID(idPart)
	if err != nil {
		return StreamRef{}, err
	}

	return StreamRef{Type: streamType, ID: streamID}, nil
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
//nolint:exhaustruct_v5 // zero-value proves interface is satisfied at compile time
var _ fmt.Stringer = StreamRef{}
