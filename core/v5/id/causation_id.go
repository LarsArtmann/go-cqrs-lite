package id

// CausationMarker is a phantom type for branding CausationIDs.
// art-dupl:accept ADR-0152 v5 copy-forward twin of the v4 train; removed with v4 in T26
type CausationMarker struct{}

// CausationID is a strongly-typed identifier for causation tracking.
// Use this to ensure type safety when working with causation IDs.
type CausationID = Of[CausationMarker]

// NewCausationID generates a new random CausationID.
func NewCausationID() CausationID {
	return New[CausationMarker]()
}

// ParseCausationID converts a string to a CausationID.
func ParseCausationID(s string) (CausationID, error) {
	return Parse[CausationMarker](s)
}
