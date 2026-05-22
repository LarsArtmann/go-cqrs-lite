package event

import (
	"fmt"
	"sort"
	"sync"
)

type upcasterRegistry struct {
	mu        sync.RWMutex
	upcasters map[Type][]upcaster
}

func newUpcasterRegistry() *upcasterRegistry {
	return &upcasterRegistry{
		upcasters: make(map[Type][]upcaster),
		mu:        sync.RWMutex{},
	}
}

func (r *upcasterRegistry) register(u upcaster) {
	r.mu.Lock()
	defer r.mu.Unlock()

	eventType := u.SourceType()
	r.upcasters[eventType] = append(r.upcasters[eventType], u)

	sort.Slice(r.upcasters[eventType], func(i, j int) bool {
		return r.upcasters[eventType][i].SourceVersion() < r.upcasters[eventType][j].SourceVersion()
	})
}

func (r *upcasterRegistry) upcast(evt Event) (Event, error) {
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
			return nil, fmt.Errorf(
				"upcast %s from schema version %d: %w",
				uc.SourceType(),
				uc.SourceVersion(),
				err,
			)
		}

		next.schemaVersion = uc.SourceVersion() + 1
		current = next
	}

	return current, nil
}
