package schema

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event"
)

type VersionedStore struct {
	Store    event.Store
	registry *upcasterRegistry
}

func NewVersionedStore(store event.Store, upcasters ...Upcaster) *VersionedStore {
	reg := newUpcasterRegistry()
	for _, u := range upcasters {
		reg.register(u)
	}

	return &VersionedStore{Store: store, registry: reg}
}

func (s *VersionedStore) Load(
	ctx context.Context,
	ref event.AggregateRef,
) ([]event.Event, error) {
	events, err := s.Store.Load(ctx, ref)
	if err != nil {
		return nil, err
	}

	return s.upcastAll(events)
}

func (s *VersionedStore) LoadFromVersion(
	ctx context.Context,
	ref event.AggregateRef,
	version event.Version,
) ([]event.Event, error) {
	events, err := s.Store.LoadFromVersion(ctx, ref, version)
	if err != nil {
		return nil, err
	}

	return s.upcastAll(events)
}

func (s *VersionedStore) upcastAll(events []event.Event) ([]event.Event, error) {
	result := make([]event.Event, len(events))
	for i, evt := range events {
		upcasted, err := s.registry.upcast(evt)
		if err != nil {
			return nil, event.WrapCorruption(
				err,
				"event.upcast_failed",
				"upcast event "+evt.ID().String(),
			)
		}
		result[i] = upcasted
	}
	return result, nil
}
