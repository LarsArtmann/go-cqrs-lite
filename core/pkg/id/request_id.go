package id

// requestMarker is a phantom type for branding RequestIDs.
type requestMarker struct{}

// RequestID is a strongly-typed identifier for HTTP requests.
// Use this to ensure type safety when working with request IDs.
type RequestID = Of[requestMarker]

// NewRequestID generates a new random RequestID.
func NewRequestID() RequestID {
	return New[requestMarker]()
}

// ParseRequestID converts a string to a RequestID.
func ParseRequestID(s string) (RequestID, error) {
	return Parse[requestMarker](s)
}

// MustParseRequestID converts a string to a RequestID, panicking on error.
func MustParseRequestID(s string) RequestID {
	return MustParse[requestMarker](s)
}
