package event

import (
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// StreamKey returns the canonical key for an event stream,
// combining aggregate type and aggregate ID.
func StreamKey(aggregateType AggregateType, aggregateID id.AggregateID) string {
	return string(aggregateType) + ":" + aggregateID.String()
}
