package sync

import (
	"encoding/json"
	"time"
)

// OperationType represents the kind of sync operation.
type OperationType string

const (
	// OpCreate indicates a new entity was created.
	OpCreate OperationType = "create"
	// OpUpdate indicates an existing entity was modified.
	OpUpdate OperationType = "update"
	// OpDelete indicates an entity was removed.
	OpDelete OperationType = "delete"
)

// Operation represents a single sync operation with a generic payload.
// Operations are the fundamental unit of change in an operation-based sync system.
//
// Type parameter T is the payload type (e.g., a domain entity).
type Operation[T any] struct {
	// ID is the unique identifier for this operation.
	ID OperationID `json:"id"`
	// Type classifies the operation (create, update, delete).
	Type OperationType `json:"type"`
	// NodeID identifies which node produced this operation.
	NodeID NodeID `json:"nodeId"`
	// Timestamp is when the operation was created.
	Timestamp time.Time `json:"timestamp"`
	// VectorClock captures the causal context at operation creation time.
	VectorClock VectorClock `json:"vectorClock"`
	// Payload contains the entity data for this operation.
	Payload T `json:"payload"`
}

// NewOperation creates a new operation with the given parameters.
func NewOperation[T any](
	id OperationID,
	opType OperationType,
	nodeID NodeID,
	payload T,
) *Operation[T] {
	return &Operation[T]{
		ID:          id,
		Type:        opType,
		NodeID:      nodeID,
		Timestamp:   time.Now().UTC(),
		Payload:     payload,
		VectorClock: NewVectorClock(),
	}
}

// Serialize converts the operation to JSON bytes.
func (op *Operation[T]) Serialize() ([]byte, error) {
	return json.Marshal(op) //nolint:wrapcheck // implementing standard serialization
}

// DeserializeOperation parses JSON bytes into an Operation[T].
func DeserializeOperation[T any](data []byte) (*Operation[T], error) {
	var op Operation[T]

	err := json.Unmarshal(data, &op)
	if err != nil {
		return nil, err //nolint:wrapcheck // implementing standard deserialization
	}

	return &op, nil
}
