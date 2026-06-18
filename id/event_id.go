package id

// EventMarker is a phantom type for branding EventIDs.
type EventMarker struct{}

// EventID is a strongly-typed identifier for domain events.
// Use this to ensure type safety when working with event IDs.
type EventID = Of[EventMarker]

// NewEventID generates a new random EventID.
func NewEventID() EventID {
	return New[EventMarker]()
}

// ParseEventID converts a string to an EventID.
func ParseEventID(s string) (EventID, error) {
	return Parse[EventMarker](s)
}
