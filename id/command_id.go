package id

type commandMarker struct{}

// CommandID is a branded unique identifier for command messages.
type CommandID = Of[commandMarker]

// NewCommandID generates a new unique CommandID.
func NewCommandID() CommandID {
	return New[commandMarker]()
}

// ParseCommandID parses a string into a CommandID.
// Returns an error if the string is not a valid ULID.
func ParseCommandID(s string) (CommandID, error) {
	return Parse[commandMarker](s)
}

// MustParseCommandID parses a string into a CommandID.
// Panics if the string is not a valid ULID.
func MustParseCommandID(s string) CommandID {
	return MustParse[commandMarker](s)
}
