package saga

import (
	"context"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// MemoryStore is an in-memory implementation of Store for testing.
type MemoryStore struct {
	mu     sync.RWMutex
	states map[string]*State
}

var _ Store = (*MemoryStore)(nil)

// NewMemoryStore creates a new in-memory saga store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		states: make(map[string]*State),
	}
}

// Save creates or updates a saga state.
func (s *MemoryStore) Save(_ context.Context, state *State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if state == nil {
		return event.WrapRejection(ErrSagaNotFound, "saga.nil_state", "state is nil")
	}

	s.states[state.ID.String()] = copyState(state)

	return nil
}

// Load retrieves a saga state by ID.
func (s *MemoryStore) Load(_ context.Context, id id.AggregateID) (*State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.states[id.String()]
	if !ok {
		return nil, event.WrapRejection(ErrSagaNotFound, "saga.not_found", "saga "+id.String()+" not found")
	}

	return copyState(state), nil
}

// LoadAllRunning returns all saga states that are currently running or compensating.
func (s *MemoryStore) LoadAllRunning(_ context.Context) ([]*State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var running []*State

	for _, state := range s.states {
		if state.Status == StatusRunning || state.Status == StatusCompensating {
			running = append(running, copyState(state))
		}
	}

	return running, nil
}

// copyState creates a defensive copy of a saga state.
func copyState(s *State) *State {
	if s == nil {
		return nil
	}

	cp := *s

	return &cp
}
