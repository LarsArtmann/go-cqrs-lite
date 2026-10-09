package id

// CorrelationMarker is a phantom type for branding CorrelationIDs.
// Exported so domain packages can create domain-specific IDs interoperable
// with CorrelationID and integrate type-parameterized tooling (e.g. BrandNamer).
type CorrelationMarker struct{}

// CorrelationID is a strongly-typed identifier for distributed tracing correlation.
// Use this to ensure type safety when working with correlation IDs.
type CorrelationID = Of[CorrelationMarker]

// NewCorrelationID generates a new random CorrelationID.
func NewCorrelationID() CorrelationID {
	return New[CorrelationMarker]()
}

// ParseCorrelationID converts a string to a CorrelationID.
func ParseCorrelationID(s string) (CorrelationID, error) {
	return Parse[CorrelationMarker](s)
}
