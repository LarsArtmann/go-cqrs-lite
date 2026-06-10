package id

// userMarker is a phantom type for branding UserIDs.
type userMarker struct{}

// UserID is a strongly-typed identifier for users.
// Use this to ensure type safety when working with user IDs.
type UserID = Of[userMarker]

// NewUserID generates a new random UserID.
func NewUserID() UserID {
	return New[userMarker]()
}

// ParseUserID converts a string to a UserID.
func ParseUserID(s string) (UserID, error) {
	return Parse[userMarker](s)
}
