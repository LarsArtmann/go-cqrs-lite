package id

// RequestMarker is a phantom type for branding RequestIDs.
// Exported so domain packages can create domain-specific IDs interoperable
// with RequestID and integrate type-parameterized tooling (e.g. BrandNamer).
type RequestMarker struct{}

// RequestID is a strongly-typed identifier for HTTP requests.
// Use this to ensure type safety when working with request IDs.
type RequestID = Of[RequestMarker]

// NewRequestID generates a new random RequestID.
func NewRequestID() RequestID {
	return New[RequestMarker]()
}

// ParseRequestID converts a string to a RequestID.
func ParseRequestID(s string) (RequestID, error) {
	return Parse[RequestMarker](s)
}
