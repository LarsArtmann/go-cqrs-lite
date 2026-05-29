package command

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
)

type AggregateType = event.AggregateType

type AggregateRef = event.AggregateRef

func ParseAggregateType(s string) (AggregateType, error) {
	return event.ParseAggregateType(s)
}

func MustParseAggregateType(s string) AggregateType {
	t, err := event.ParseAggregateType(s)
	if err != nil {
		panic(fmt.Sprintf("command.MustParseAggregateType: %v", err))
	}

	return t
}

func NewAggregateRef(aggregateType AggregateType, aggregateID id.AggregateID) AggregateRef {
	return event.NewAggregateRef(aggregateType, aggregateID)
}
