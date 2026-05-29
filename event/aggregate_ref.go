package event

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/id"
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

// Verify AggregateRef satisfies fmt.Stringer at compile time.
var _ fmt.Stringer = AggregateRef{}
