package event

import (
	"context"
)

// VersionedStore wraps an event.Store and automatically upcasts loaded events.
type VersionedStore struct {
	Store
	registry *upcasterRegistry
}

// NewVersionedStore creates a store that applies registered upcasters
// when loading events. Pass nil or no upcasters for a no-op wrapper.
func NewVersionedStore(store Store, upcasters ...Upcaster) *VersionedStore {
	reg := newUpcasterRegistry()
	for _, u := range upcasters {
		reg.register(u)
	}

	return &VersionedStore{Store: store, registry: reg}
}

// Load retrieves and upcasts events for an aggregate.
func (s *VersionedStore) Load(
	ctx context.Context,
	ref AggregateRef,
) ([]Event, error) {
	events, err := s.Store.Load(ctx, ref)
	if err != nil {
		return nil, err
	}

	return s.upcastAll(events)
}

// LoadFromVersion retrieves and upcasts events starting from a specific version.
func (s *VersionedStore) LoadFromVersion(
	ctx context.Context,
	ref AggregateRef,
	version Version,
) ([]Event, error) {
	events, err := s.Store.LoadFromVersion(ctx, ref, version)
	if err != nil {
		return nil, err
	}

	return s.upcastAll(events)
}

func (s *VersionedStore) upcastAll(events []Event) ([]Event, error) {
	result := make([]Event, len(events))
	for i, evt := range events {
		upcasted, err := s.registry.upcast(evt)
		if err != nil {
			return nil, WrapCorruption(
				err,
				"event.upcast_failed",
				"upcast event "+evt.ID().String(),
			)
		}
		result[i] = upcasted
	}
	return result, nil
}
