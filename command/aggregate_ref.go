package command

import (
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// AggregateType and AggregateRef are type aliases for the id package types.
// These exist so command consumers can import everything from command/
// without adding a direct event/ dependency for these core identifiers.
// This is an intentional convenience re-export, not a layering violation:
// commands operate on the same aggregate identity as events.
type AggregateType = id.AggregateType

type AggregateRef = id.AggregateRef

func ParseAggregateType(s string) (AggregateType, error) {
	t, err := id.ParseAggregateType(s)
	if err != nil {
		return "", errorfamily.WrapRejection(
			err,
			"command.parse_aggregate_type",
			"parse aggregate type",
		)
	}

	return t, nil
}

func NewAggregateRef(aggregateType AggregateType, aggregateID id.AggregateID) AggregateRef {
	return id.NewAggregateRef(aggregateType, aggregateID)
}
