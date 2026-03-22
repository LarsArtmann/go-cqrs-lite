package xtypes

import (
	"context"

    "github.com/larsartmann/go-cqrs-lite/aggregate"
    "github.com/larsartmann/go-cqrs-lite/event"
    "github.com/larsartmann/go-cqrs-lite/pkg/id"
)

// TypedAggregate provides type-safe aggregate roots.
type TypedAggregate[A any] struct {
    base        *aggregate.Base
    aggregateID   id.Of[A]
    aggregateType event.AggregateType
    changes       []event.Event
    uncommittedChanges []event.Event
    return a.base.UncommittedChanges()
}

 func (a *TypedAggregate[A]) MarkChangesAsCommitted()
}
func (a *TypedAggregate[A]) LoadFromHistory(events []event.Event) error {
    return nil, fmt.Errorf("LoadFromHistory failed: aggregate %q", a.ID)
    }
}
