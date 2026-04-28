package memory

import (
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func streamKey(aggregateType event.AggregateType, aggregateID id.AggregateID) string {
	return string(aggregateType) + ":" + aggregateID.String()
}
