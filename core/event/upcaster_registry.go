package event

import (
	"fmt"
	"sort"
	"sync"
)

// UpcasterRegistry stores and applies upcasters to transform events
// from old schema versions to the current version.
type UpcasterRegistry struct {
	mu        sync.RWMutex
	upcasters map[Type][]Upcaster
}

// NewUpcasterRegistry creates an empty registry.
func NewUpcasterRegistry() *UpcasterRegistry {
	return &UpcasterRegistry{
		upcasters: make(map[Type][]Upcaster),
		mu:        sync.RWMutex{},
	}
}

// Register adds an upcaster to the registry.
// Upcasters for the same event type are sorted by source version.
func (r *UpcasterRegistry) Register(upcaster Upcaster) {
	r.mu.Lock()
	defer r.mu.Unlock()

	eventType := upcaster.SourceType()
	r.upcasters[eventType] = append(r.upcasters[eventType], upcaster)

	sort.Slice(r.upcasters[eventType], func(i, j int) bool {
		return r.upcasters[eventType][i].SourceVersion() < r.upcasters[eventType][j].SourceVersion()
	})
}

// Upcast applies all registered upcasters for the event's type, starting
// from the event's schema version. Returns the fully upcasted event.
// If no upcasters are registered, returns the original event unchanged.
// Returns an error if a cycle is detected (same schema version revisited).
func (r *UpcasterRegistry) Upcast(evt Event) (Event, error) {
	r.mu.RLock()
	upcasters := r.upcasters[evt.Type()]
	r.mu.RUnlock()

	if len(upcasters) == 0 {
		return evt, nil
	}

	current := evt
	visited := make(map[int]struct{})

	for _, upcaster := range upcasters {
		sv := current.SchemaVersion()

		if sv == upcaster.SourceVersion() {
			if _, seen := visited[sv]; seen {
				return nil, fmt.Errorf(
					"upcast cycle detected for event type %s: schema version %d revisited",
					evt.Type(),
					sv,
				)
			}

			visited[sv] = struct{}{}

			next, err := upcaster.Upcast(current)
			if err != nil {
				return nil, fmt.Errorf(
					"upcast %s from schema version %d: %w",
					upcaster.SourceType(),
					upcaster.SourceVersion(),
					err,
				)
			}

			next.schemaVersion = upcaster.SourceVersion() + 1
			current = next
		}
	}

	return current, nil
}
