package id

// CommandMarker is a phantom type for branding CommandIDs.
type CommandMarker struct{}

// CommandID is a branded unique identifier for command messages.
type CommandID = Of[CommandMarker]

// NewCommandID generates a new unique CommandID.
func NewCommandID() CommandID {
	return New[CommandMarker]()
}

// ParseCommandID parses a string into a CommandID.
// Returns an error if the string is not a valid ULID.
func ParseCommandID(s string) (CommandID, error) {
	return Parse[CommandMarker](s)
}
