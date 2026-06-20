package id

// TransactionMarker is a phantom type for branding TransactionIDs.
// A TransactionID tracks cross-aggregate consistency boundaries — all events
// produced within a single logical transaction share the same TransactionID.
type TransactionMarker struct{}

// TransactionID is a strongly-typed identifier for cross-aggregate
// transaction tracking. Use this to correlate events that must be consumed
// atomically across multiple aggregates or projection handlers.
type TransactionID = Of[TransactionMarker]

// NewTransactionID generates a new random TransactionID.
func NewTransactionID() TransactionID {
	return New[TransactionMarker]()
}

// ParseTransactionID converts a string to a TransactionID.
func ParseTransactionID(s string) (TransactionID, error) {
	return Parse[TransactionMarker](s)
}
