package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
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

func (s *ReadModelStore) Handle(_ context.Context, evt event.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	aggID := evt.AggregateID()
	codec := event.JSONCodec{}

	switch evt.Type() {
	case eventUserCreated:
		p, err := event.DecodePayload[UserCreatedPayload](evt, codec)
		if err != nil {
			return fmt.Errorf("decode UserCreated in projection: %w", err)
		}

		s.users[aggID] = ReadModel(p)
	case eventUserNameChanged:
		p, err := event.DecodePayload[UserNameChangedPayload](evt, codec)
		if err != nil {
			return fmt.Errorf("decode UserNameChanged in projection: %w", err)
		}

		if existing, ok := s.users[aggID]; ok {
			existing.Name = p.Name
			s.users[aggID] = existing
		}
	}

	return nil
}
