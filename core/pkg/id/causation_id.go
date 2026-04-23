package id

// causationMarker is a phantom type for branding CausationIDs.
type causationMarker struct{}

// CausationID is a strongly-typed identifier for causation tracking.
// Use this to ensure type safety when working with causation IDs.
type CausationID = Of[causationMarker]

// NewCausationID generates a new random CausationID.
func NewCausationID() CausationID {
	return New[causationMarker]()
}

// ParseCausationID converts a string to a CausationID.
func ParseCausationID(s string) (CausationID, error) {
	return Parse[causationMarker](s)
}

// MustParseCausationID converts a string to a CausationID, panicking on error.
func MustParseCausationID(s string) CausationID {
	return MustParse[causationMarker](s)
}
