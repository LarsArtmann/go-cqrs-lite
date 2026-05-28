package event

import (
	"sort"
	"strconv"
	"sync"
)

type upcasterRegistry struct {
	mu        sync.RWMutex
	upcasters map[Type][]Upcaster
}

func newUpcasterRegistry() *upcasterRegistry {
	return &upcasterRegistry{
		upcasters: make(map[Type][]Upcaster),
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
			return nil, WrapCorruption(
				err,
				"event.upcast_failed",
				"upcast "+string(uc.SourceType())+" from schema version "+strconv.Itoa(int(uc.SourceVersion())),
			)
		}

		next.schemaVersion = uc.SourceVersion() + 1
		current = next
	}

	return current, nil
}
