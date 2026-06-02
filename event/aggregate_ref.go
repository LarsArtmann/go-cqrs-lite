package event

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// AggregateRef uniquely identifies an aggregate instance by its type and ID.
// Use this to pass aggregate identity as a single value instead of separate params.
type AggregateRef struct {
	Type AggregateType
	ID   id.AggregateID
}

func (r AggregateRef) String() string {
	return r.Type.String() + ":" + r.ID.String()
}

// StreamKey returns the canonical key for an event stream.
func (r AggregateRef) StreamKey() string {
	return r.String()
}

// NewAggregateRef creates an AggregateRef from a type and ID.
func NewAggregateRef(aggregateType AggregateType, aggregateID id.AggregateID) AggregateRef {
	return AggregateRef{Type: aggregateType, ID: aggregateID}
}

// IsZero returns true if both Type and ID are their zero values.
func (r AggregateRef) IsZero() bool {
	return r.Type == "" && r.ID.IsZero()
}

// Validate returns an error if Type is empty or ID is zero.
func (r AggregateRef) Validate() error {
	if r.Type == "" {
		return ErrEmptyAggregateType
	}

	if r.ID.IsZero() {
		return ErrNilAggregateID
	}

	return nil
}

// Verify AggregateRef satisfies fmt.Stringer at compile time.
var _ fmt.Stringer = AggregateRef{}
