package schema

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event"
)

type VersionedStore struct {
	event.Store

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
		return nil, fmt.Errorf("versioned store load %s: %w", ref, err)
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
		return nil, fmt.Errorf("versioned store load from version %s@%s: %w", ref, version, err)
	}

	return s.upcastAll(events)
}

func (s *VersionedStore) LoadToVersion(
	ctx context.Context,
	ref event.AggregateRef,
	maxVersion event.Version,
) ([]event.Event, error) {
	events, err := s.Store.LoadToVersion(ctx, ref, maxVersion)
	if err != nil {
		return nil, fmt.Errorf("versioned store load to version %s@%s: %w", ref, maxVersion, err)
	}

	return s.upcastAll(events)
}

func (s *VersionedStore) LoadToTimestamp(
	ctx context.Context,
	ref event.AggregateRef,
	maxTime time.Time,
) ([]event.Event, error) {
	events, err := s.Store.LoadToTimestamp(ctx, ref, maxTime)
	if err != nil {
		return nil, fmt.Errorf(
			"versioned store load to timestamp %s@%s: %w",
			ref,
			maxTime.Format(time.RFC3339),
			err,
		)
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
				"schema.upcast_failed",
				"upcast event "+evt.ID().String(),
			)
		}

		result[i] = upcasted
	}

	return result, nil
}
