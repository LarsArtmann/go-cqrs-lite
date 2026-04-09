package id

// AggregateMarker is a phantom type for branding AggregateIDs.
// Export it so domain packages can create domain-specific IDs interoperable with AggregateID.
type AggregateMarker struct{}

// AggregateID is a strongly-typed identifier for aggregate roots.
// Use this to ensure type safety when working with aggregate IDs.
type AggregateID = Of[AggregateMarker]

// NewAggregateID generates a new random AggregateID.
func NewAggregateID() AggregateID {
	return New[AggregateMarker]()
}

// ParseAggregateID converts a string to an AggregateID.
func ParseAggregateID(s string) (AggregateID, error) {
	return Parse[AggregateMarker](s)
}

// MustParseAggregateID converts a string to an AggregateID, panicking on error.
func MustParseAggregateID(s string) AggregateID {
	return MustParse[AggregateMarker](s)
}
