package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

type ReadModel struct {
	Email string
	Name  string
}

type ReadModelStore struct {
	mu    sync.RWMutex
	users map[id.AggregateID]ReadModel
}

func NewReadModelStore() *ReadModelStore {
	return &ReadModelStore{users: make(map[id.AggregateID]ReadModel)}
}

func (s *ReadModelStore) Get(aggID id.AggregateID) (ReadModel, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m, ok := s.users[aggID]

	return m, ok
}

func (s *ReadModelStore) List() []ReadModel {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]ReadModel, 0, len(s.users))
	for _, m := range s.users {
		result = append(result, m)
	}

	return result
}

func (s *ReadModelStore) Name() string { return "user-read-model" }

func (s *ReadModelStore) EventTypes() []event.Type {
	return []event.Type{eventUserCreated, eventUserNameChanged, eventUserDeleted, eventUserReborn}
}

func (s *ReadModelStore) Handle(_ context.Context, evt event.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	aggID := evt.AggregateID()
	c := codec.JSONCodec{}

	switch evt.Type() {
	case eventUserCreated:
		p, err := event.DecodePayload[UserCreatedPayload](evt, c)
		if err != nil {
			return fmt.Errorf("decode UserCreated in projection: %w", err)
		}

		s.users[aggID] = ReadModel(p)
	case eventUserNameChanged:
		p, err := event.DecodePayload[UserNameChangedPayload](evt, c)
		if err != nil {
			return fmt.Errorf("decode UserNameChanged in projection: %w", err)
		}

		if existing, ok := s.users[aggID]; ok {
			existing.Name = p.Name
			s.users[aggID] = existing
		}
	case eventUserDeleted:
		delete(s.users, aggID)
	case eventUserReborn:
		p, err := event.DecodePayload[UserRebornPayload](evt, c)
		if err != nil {
			return fmt.Errorf("decode UserReborn in projection: %w", err)
		}

		s.users[aggID] = ReadModel(p)
	}

	return nil
}

var _ event.Projection = (*ReadModelStore)(nil)
