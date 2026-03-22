package id

// aggregateMarker is a phantom type for branding AggregateIDs.
type aggregateMarker struct{}

// AggregateID is a strongly-typed identifier for aggregate roots.
// Use this to ensure type safety when working with aggregate IDs.
type AggregateID = Of[aggregateMarker]

// NewAggregateID generates a new random AggregateID.
func NewAggregateID() AggregateID {
	return New[aggregateMarker]()
}

// ParseAggregateID converts a string to an AggregateID.
func ParseAggregateID(s string) (AggregateID, error) {
	return Parse[aggregateMarker](s)
}

// MustParseAggregateID converts a string to an AggregateID, panicking on error.
func MustParseAggregateID(s string) AggregateID {
	return MustParse[aggregateMarker](s)
}
