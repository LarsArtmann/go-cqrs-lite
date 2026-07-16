package decider

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// StrictApply wraps an Apply function to return an error when an event type
// is not in the knownTypes list. This prevents silent data corruption when
// a fold function's switch statement doesn't handle a new event type.
//
// Usage:
//
//	d := decider.Decider[State]{
//	    Initial: State{},
//	    Apply: decider.StrictApply(fold, []event.Type{
//	        "user.created",
//	        "user.updated",
//	        "user.deleted",
//	    }),
//	}
//
// If Apply is called with an event type not in knownTypes, it returns:
//
//	fmt.Errorf("strict-apply: unknown event type: %s", evt.Type())
//
// This is the library-level fix for rule C003 (silent-unknown-event-fold).
func StrictApply[State any](
	apply func(state State, evt event.Event) (State, error),
	knownTypes []event.Type,
) func(state State, evt event.Event) (State, error) {
	known := make(map[event.Type]bool, len(knownTypes))
	for _, t := range knownTypes {
		known[t] = true
	}

	return func(state State, evt event.Event) (State, error) {
		if !known[evt.Type()] {
			return state, fmt.Errorf("strict-apply: unknown event type: %s", evt.Type())
		}

		return apply(state, evt)
	}
}
