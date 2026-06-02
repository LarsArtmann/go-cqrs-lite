package projection

import (
	"slices"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

func subscribesTo(p event.Projection, eventType event.Type) bool {
	types := p.EventTypes()

	return len(types) == 0 || slices.Contains(types, eventType)
}

func filterByEventTypes(events []event.Event, types []event.Type) []event.Event {
	if len(types) == 0 {
		return events
	}

	result := make([]event.Event, 0, len(events))

	for _, evt := range events {
		if slices.Contains(types, evt.Type()) {
			result = append(result, evt)
		}
	}

	return result
}

func filterFromCheckpoint(
	all []event.Event,
	types []event.Type,
	checkpoint event.Checkpoint,
) []event.Event {
	result := make([]event.Event, 0, len(all))

	pastCheckpoint := checkpoint.IsZero()

	for _, evt := range all {
		if !pastCheckpoint {
			if evt.ID() == checkpoint.EventID {
				pastCheckpoint = true
			}

			continue
		}

		if len(types) > 0 && !slices.Contains(types, evt.Type()) {
			continue
		}

		result = append(result, evt)
	}

	return result
}
