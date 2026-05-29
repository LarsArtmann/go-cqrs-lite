package id

// clientMarker is a phantom type for branding ClientIDs.
type clientMarker struct{}

// ClientID is a strongly-typed identifier for the client device that created an event.
// Use this for offline-first attribution and conflict detection.
type ClientID = Of[clientMarker]

// NewClientID generates a new random ClientID.
func NewClientID() ClientID {
	return New[clientMarker]()
}

// ParseClientID converts a string to a ClientID.
func ParseClientID(s string) (ClientID, error) {
	return Parse[clientMarker](s)
}

// MustParseClientID converts a string to a ClientID, panicking on error.
func MustParseClientID(s string) ClientID {
	return MustParse[clientMarker](s)
}
