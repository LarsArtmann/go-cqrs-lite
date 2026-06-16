package command

import (
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

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
