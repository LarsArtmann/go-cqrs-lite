package schema

import (
	"sort"
	"strconv"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
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

// newUpcasterRegistryFrom creates a registry and registers all non-nil upcasters.
func newUpcasterRegistryFrom(upcasters []Upcaster) *upcasterRegistry {
	reg := newUpcasterRegistry()

	for _, u := range upcasters {
		if u != nil {
			reg.register(u)
		}
	}

	return reg
}

// register adds an upcaster. Duplicate (source type, source version)
// registrations are ignored — the FIRST registration wins, keeping the
// chain deterministic.
func (r *upcasterRegistry) register(upcaster Upcaster) {
	r.mu.Lock()
	defer r.mu.Unlock()

	eventType := upcaster.SourceType()

	for _, existing := range r.upcasters[eventType] {
		if existing.SourceVersion() == upcaster.SourceVersion() {
			return
		}
	}

	r.upcasters[eventType] = append(r.upcasters[eventType], upcaster)

	sort.SliceStable(r.upcasters[eventType], func(i, j int) bool {
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

		if next == nil || next == current {
			return nil, errorfamily.WrapCorruption(
				ErrInvalidUpcastResult,
				"schema.invalid_upcast_result",
				"upcast "+string(uc.SourceType())+" from schema version "+strconv.Itoa(
					int(uc.SourceVersion()),
				),
			)
		}

		// Preserve an upcaster-stamped version (a v1→v3 jump keeps v3):
		// only stamp source+1 when the result does not already advance past
		// the source version (a fresh event defaults to schema version 1).
		if next.SchemaVersion() <= uc.SourceVersion() {
			event.WithSchemaVersion(uc.SourceVersion().Increment())(next)
		}

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
