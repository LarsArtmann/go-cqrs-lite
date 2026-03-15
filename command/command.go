// Package command provides the Command types and dispatcher infrastructure for CQRS applications.
//
// Key design principles:
//   - Type-safe command definitions
//   - Command dispatcher with handler registration
//   - Context-aware operations
//   - No panics, explicit error handling
//
// Reference: ChastityAPI, Cyberdom patterns
// HOW_TO_GOLANG.md coding standards
//   - Max 250 lines per file
//   - Max 30 lines per function
//   - No `any` types
//   - Context as first parameter
//   - Sentinels for common error states
//   - No external dependencies (except google/uuid, cockroachdb/errors)

// - Files under 250 lines
// - All exported types have Go doc comments
// - Use errors.Is for error comparison (not assertions)
// - Use package names that match domain (e.g., "command", not "commands")
package command

import (
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
)

// Command represents a write operation in CQRS
type Command interface {
	ID() string
	Type() CommandType
	AggregateID() string
	Timestamp() time.Time
}

// CommandType is a type identifier for commands
type CommandType string

// BaseCommand provides a default implementation of Command interface
type BaseCommand struct {
	id          string
	cmdType     CommandType
	aggregateID string
	timestamp   time.Time
}

func (c *BaseCommand) ID() string           { return c.id }
func (c *BaseCommand) Type() CommandType    { return c.cmdType }
func (c *BaseCommand) AggregateID() string  { return c.aggregateID }
func (c *BaseCommand) Timestamp() time.Time { return c.timestamp }

// NewCommand creates a new command with validation
func NewCommand(
	cmdType CommandType,
	aggregateID string,
	opts ...CommandOption,
) (*BaseCommand, error) {
	if aggregateID == "" {
		return nil, errors.New("aggregate ID is required")
	}
	if cmdType == "" {
		return nil, errors.New("command type is required")
	}

	cmd := &BaseCommand{
		id:          uuid.New().String(),
		cmdType:     cmdType,
		aggregateID: aggregateID,
		timestamp:   time.Now(),
	}
	for _, opt := range opts {
		opt(cmd)
	}
	return cmd, nil
}

// CommandOption is a functional option for configuring command creation
type CommandOption func(*BaseCommand)

// WithCommandID sets a custom command ID (for idempotency)
func WithCommandID(id string) CommandOption {
	return func(c *BaseCommand) { c.id = id }
}

// WithTimestamp sets a custom timestamp
func WithTimestamp(t time.Time) CommandOption {
	return func(c *BaseCommand) { c.timestamp = t }
}
