package main

import (
	"context"
	"encoding/json"
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

func (s *ReadModelStore) unmarshalPayload(evt event.Event, payload any) error {
	if err := json.Unmarshal(evt.Payload(), payload); err != nil {
		return fmt.Errorf("unmarshal %s in projection: %w", evt.Type(), err)
	}

	return nil
}

func (s *ReadModelStore) Handle(_ context.Context, evt event.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	aggID := evt.AggregateID()

	switch evt.Type() {
	case eventUserCreated:
		var p UserCreatedPayload
		if err := s.unmarshalPayload(evt, &p); err != nil {
			return err
		}

		s.users[aggID] = ReadModel{Email: p.Email, Name: p.Name}
	case eventUserNameChanged:
		var p UserNameChangedPayload
		if err := s.unmarshalPayload(evt, &p); err != nil {
			return err
		}

		if existing, ok := s.users[aggID]; ok {
			existing.Name = p.Name
			s.users[aggID] = existing
		}
	}

	return nil
}
