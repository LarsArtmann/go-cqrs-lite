package stream

import (
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	ro "github.com/samber/ro"
)

// Operator is a function that transforms one Observable into another.
type Operator func(ro.Observable[event.Event]) ro.Observable[event.Event]

// FilterByType returns an operator that only passes events matching the given types.
// If types is empty, all events pass through.
func FilterByType(types ...event.Type) Operator {
	if len(types) == 0 {
		return func(src ro.Observable[event.Event]) ro.Observable[event.Event] { return src }
	}

	set := make(map[event.Type]bool, len(types))

	for _, t := range types {
		set[t] = true
	}

	return ro.Filter[event.Event](func(evt event.Event) bool {
		return set[evt.Type()]
	})
}

// FilterFromCheckpoint returns an operator that drops events until the
// checkpoint event ID is seen (exclusive). If checkpoint is zero, all events pass.
func FilterFromCheckpoint(checkpoint id.EventID) Operator {
	if checkpoint.IsZero() {
		return func(src ro.Observable[event.Event]) ro.Observable[event.Event] { return src }
	}

	passed := false

	return ro.Filter[event.Event](func(evt event.Event) bool {
		if passed {
			return true
		}

		if evt.ID() == checkpoint {
			passed = true
		}

		return false
	})
}
