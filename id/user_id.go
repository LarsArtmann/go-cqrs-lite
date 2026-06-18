package id

// UserMarker is a phantom type for branding UserIDs.
// Exported so domain packages can create domain-specific IDs interoperable
// with UserID and integrate type-parameterized tooling (e.g. BrandNamer).
type UserMarker struct{}

// UserID is a strongly-typed identifier for users.
// Use this to ensure type safety when working with user IDs.
type UserID = Of[UserMarker]

// NewUserID generates a new random UserID.
func NewUserID() UserID {
	return New[UserMarker]()
}

// ParseUserID converts a string to a UserID.
func ParseUserID(s string) (UserID, error) {
	return Parse[UserMarker](s)
}
