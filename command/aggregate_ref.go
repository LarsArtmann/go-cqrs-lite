package command

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/id"
)

type AggregateType string

func (a AggregateType) String() string { return string(a) }

func (a AggregateType) IsZero() bool { return a == "" }

func ParseAggregateType(s string) (AggregateType, error) {
	if s == "" {
		return "", ErrEmptyAggregateType
	}

	return AggregateType(s), nil
}

func MustParseAggregateType(s string) AggregateType {
	t, err := ParseAggregateType(s)
	if err != nil {
		panic(fmt.Sprintf("command.MustParseAggregateType: %v", err))
	}

	return t
}

type AggregateRef struct {
	Type AggregateType
	ID   id.AggregateID
}

func (r AggregateRef) String() string {
	return r.Type.String() + ":" + r.ID.String()
}

func NewAggregateRef(aggregateType AggregateType, aggregateID id.AggregateID) AggregateRef {
	return AggregateRef{Type: aggregateType, ID: aggregateID}
}
