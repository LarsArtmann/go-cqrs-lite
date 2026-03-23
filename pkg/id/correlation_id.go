package id

// correlationMarker is a phantom type for branding CorrelationIDs.
type correlationMarker struct{}

// CorrelationID is a strongly-typed identifier for distributed tracing correlation.
// Use this to ensure type safety when working with correlation IDs.
type CorrelationID = Of[correlationMarker]

// NewCorrelationID generates a new random CorrelationID.
func NewCorrelationID() CorrelationID {
	return New[correlationMarker]()
}

// ParseCorrelationID converts a string to a CorrelationID.
func ParseCorrelationID(s string) (CorrelationID, error) {
	return Parse[correlationMarker](s)
}

// MustParseCorrelationID converts a string to a CorrelationID, panicking on error.
func MustParseCorrelationID(s string) CorrelationID {
	return MustParse[correlationMarker](s)
}
