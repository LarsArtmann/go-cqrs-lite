package command

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

type AggregateType = event.AggregateType

type AggregateRef = event.AggregateRef

func ParseAggregateType(s string) (AggregateType, error) {
	t, err := event.ParseAggregateType(s)
	if err != nil {
		return "", fmt.Errorf("command.ParseAggregateType: %w", err)
	}

	return t, nil
}

func NewAggregateRef(aggregateType AggregateType, aggregateID id.AggregateID) AggregateRef {
	return event.NewAggregateRef(aggregateType, aggregateID)
}
