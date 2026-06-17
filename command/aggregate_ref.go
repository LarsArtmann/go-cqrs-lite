package command

import (
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// AggregateType and AggregateRef are type aliases for the event package types.
// These exist so command consumers can import everything from command/
// without adding a direct event/ dependency for these core identifiers.
// This is an intentional convenience re-export, not a layering violation:
// commands operate on the same aggregate identity as events.
type AggregateType = event.AggregateType

type AggregateRef = event.AggregateRef

func ParseAggregateType(s string) (AggregateType, error) {
	t, err := event.ParseAggregateType(s)
	if err != nil {
		return "", WrapRejection(err, "command.parse_aggregate_type", "parse aggregate type")
	}

	return t, nil
}

func NewAggregateRef(aggregateType AggregateType, aggregateID id.AggregateID) AggregateRef {
	return event.NewAggregateRef(aggregateType, aggregateID)
}
