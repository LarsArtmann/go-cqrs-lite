package sync

// NodeID identifies a node in the distributed sync system.
// Using a named type prevents accidental mixing with arbitrary strings.
type NodeID string

// ParseNodeID validates and creates a NodeID from a string.
// Returns an error if the string is empty.
func ParseNodeID(s string) (NodeID, error) {
	if s == "" {
		return "", ErrEmptyNodeID
	}

	return NodeID(s), nil
}

// MustParseNodeID parses a NodeID or panics.
func MustParseNodeID(s string) NodeID {
	id, err := ParseNodeID(s)
	if err != nil {
		panic(err)
	}

	return id
}

// String returns the underlying string value.
func (n NodeID) String() string { return string(n) }

// IsZero returns true if the NodeID is empty.
func (n NodeID) IsZero() bool { return n == "" }

// OperationID uniquely identifies a sync operation.
// Using a named type prevents accidental mixing with arbitrary strings.
type OperationID string

// ParseOperationID validates and creates an OperationID from a string.
// Returns an error if the string is empty.
func ParseOperationID(s string) (OperationID, error) {
	if s == "" {
		return "", ErrEmptyOperationID
	}

	return OperationID(s), nil
}

// MustParseOperationID parses an OperationID or panics.
func MustParseOperationID(s string) OperationID {
	id, err := ParseOperationID(s)
	if err != nil {
		panic(err)
	}

	return id
}

// String returns the underlying string value.
func (id OperationID) String() string { return string(id) }

// IsZero returns true if the OperationID is empty.
func (id OperationID) IsZero() bool { return id == "" }

// SyncMessageType classifies the kind of sync protocol message.
type SyncMessageType string

const (
	// SyncMessageTypeRequest represents a sync request message.
	SyncMessageTypeRequest SyncMessageType = "sync_request"
	// SyncMessageTypeResponse represents a sync response message.
	SyncMessageTypeResponse SyncMessageType = "sync_response"
)

// String returns the underlying string value.
func (t SyncMessageType) String() string { return string(t) }

// Valid returns true if the SyncMessageType is a known value.
func (t SyncMessageType) Valid() bool {
	return t == SyncMessageTypeRequest || t == SyncMessageTypeResponse
}
