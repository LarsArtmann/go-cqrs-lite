// Package xtypes provides type-safe extensions for go-cqrs-lite.
//
// This package offers strongly-typed wrappers around the core types,
// using branded identifiers to prevent mixing up IDs at compile time.
//
// Example:
//
//	type UserBrand struct{}
//	type UserID = id.Of[UserBrand]
//
//	userID := id.New[UserID]()
//	evt, _ := xtypes.NewEventBuilder("UserCreated", userID, "User", 1).Build()
package xtypes

import (
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

// Re-export ID types from pkg/id for convenience

type (
	// AggregateID is a strongly-typed identifier for aggregate roots.
	AggregateID = id.AggregateID
	// EventID is a strongly-typed identifier for domain events.
	EventID = id.EventID
	// UserID is a strongly-typed identifier for users.
	UserID = id.UserID
)

// correlationMarker is a phantom type for branding CorrelationIDs.
type correlationMarker struct{}

// CorrelationID is a strongly-typed identifier for distributed tracing.
type CorrelationID = id.Of[correlationMarker]

// NewCorrelationID generates a new random CorrelationID.
func NewCorrelationID() CorrelationID {
	return id.New[correlationMarker]()
}

// ParseCorrelationID converts a string to a CorrelationID.
func ParseCorrelationID(s string) (CorrelationID, error) {
	return id.Parse[correlationMarker](s)
}

// commandMarker is a phantom type for branding CommandIDs.
type commandMarker struct{}

// CommandID is a strongly-typed identifier for commands (for idempotency).
type CommandID = id.Of[commandMarker]

// NewCommandID generates a new random CommandID.
func NewCommandID() CommandID {
	return id.New[commandMarker]()
}

// ParseCommandID converts a string to a CommandID.
func ParseCommandID(s string) (CommandID, error) {
	return id.Parse[commandMarker](s)
}

// causationMarker is a phantom type for branding CausationIDs.
type causationMarker struct{}

// CausationID is a strongly-typed identifier for causation tracking.
type CausationID = id.Of[causationMarker]

// NewCausationID generates a new random CausationID.
func NewCausationID() CausationID {
	return id.New[causationMarker]()
}

// ParseCausationID converts a string to a CausationID.
func ParseCausationID(s string) (CausationID, error) {
	return id.Parse[causationMarker](s)
}

// requestMarker is a phantom type for branding RequestIDs.
type requestMarker struct{}

// RequestID is a strongly-typed identifier for HTTP requests.
type RequestID = id.Of[requestMarker]

// NewRequestID generates a new random RequestID.
func NewRequestID() RequestID {
	return id.New[requestMarker]()
}

// ParseRequestID converts a string to a RequestID.
func ParseRequestID(s string) (RequestID, error) {
	return id.Parse[requestMarker](s)
}
