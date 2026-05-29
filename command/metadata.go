package command

import (
	"github.com/larsartmann/go-cqrs-lite/id"
)

// Metadata contains tracing and contextual information for commands.
// Mirrors event.Metadata to enable correlation across the CQRS pipeline.
type Metadata struct {
	CorrelationID id.CorrelationID `json:"correlationId"`
	CausationID   id.CausationID   `json:"causationId"`
	UserID        id.UserID        `json:"userId"`
	RequestID     id.RequestID     `json:"requestId"`
}

// NewMetadata creates a Metadata with all fields initialized to zero values.
func NewMetadata() Metadata {
	return Metadata{
		CorrelationID: id.CorrelationID{},
		CausationID:   id.CausationID{},
		UserID:        id.UserID{},
		RequestID:     id.RequestID{},
	}
}

// Option configures command creation.
type Option func(*BasicCommand)

// WithCorrelationID sets the correlation ID for distributed tracing.
func WithCorrelationID(v id.CorrelationID) Option {
	return func(c *BasicCommand) { c.metadata.CorrelationID = v }
}

// WithCausationID sets the causation ID (indicates what triggered this command).
func WithCausationID(v id.CausationID) Option {
	return func(c *BasicCommand) { c.metadata.CausationID = v }
}

// WithUserID sets the user ID who issued the command.
func WithUserID(v id.UserID) Option {
	return func(c *BasicCommand) { c.metadata.UserID = v }
}

// WithRequestID sets the request ID for debugging.
func WithRequestID(v id.RequestID) Option {
	return func(c *BasicCommand) { c.metadata.RequestID = v }
}
