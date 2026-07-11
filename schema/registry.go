package schema

import (
	"sort"
	"strconv"
	"sync"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

type upcasterRegistry struct {
	mu        sync.RWMutex
	upcasters map[event.Type][]Upcaster
}

func newUpcasterRegistry() *upcasterRegistry {
	return &upcasterRegistry{
		upcasters: make(map[event.Type][]Upcaster),
		mu:        sync.RWMutex{},
	}
}

func (r *upcasterRegistry) register(u Upcaster) {
	r.mu.Lock()
	defer r.mu.Unlock()

	eventType := u.SourceType()
	r.upcasters[eventType] = append(r.upcasters[eventType], u)

	sort.Slice(r.upcasters[eventType], func(i, j int) bool {
		return r.upcasters[eventType][i].SourceVersion() < r.upcasters[eventType][j].SourceVersion()
	})
}

func (r *upcasterRegistry) upcast(evt event.Event) (event.Event, error) {
	r.mu.RLock()
	upcasters := r.upcasters[evt.Type()]
	r.mu.RUnlock()

	if len(upcasters) == 0 {
		return evt, nil
	}

	current := evt

	for _, uc := range upcasters {
		if current.SchemaVersion() != uc.SourceVersion() {
			continue
		}

		next, err := uc.Upcast(current)
		if err != nil {
			return nil, errorfamily.WrapCorruption(
				err,
				"event.upcast_failed",
				"upcast "+string(
					uc.SourceType(),
				)+" from schema version "+strconv.Itoa(
					int(uc.SourceVersion()),
				),
			)
		}

		event.WithSchemaVersion(uc.SourceVersion().Increment())(next)
		current = next
	}

	return current, nil
}

func (r *upcasterRegistry) upcastAll(events []event.Event) ([]event.Event, error) {
	result := make([]event.Event, len(events))
	for i, evt := range events {
		upcasted, err := r.upcast(evt)
		if err != nil {
			return nil, errorfamily.WrapCorruption(
				err,
				"schema.upcast_failed",
				"upcast event "+evt.ID().String(),
			)
		}

		result[i] = upcasted
	}

	return result, nil
}
