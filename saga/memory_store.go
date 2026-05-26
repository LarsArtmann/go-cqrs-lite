package saga

import (
	"context"
	"fmt"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// MemoryStore is an in-memory implementation of Store for testing.
type MemoryStore struct {
	mu        sync.RWMutex
	instances map[string]*Instance
}

// NewMemoryStore creates a new in-memory saga store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		instances: make(map[string]*Instance),
	}
}

// Save creates or updates a saga instance.
func (s *MemoryStore) Save(_ context.Context, instance *Instance) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if instance == nil {
		return fmt.Errorf("instance is nil: %w", ErrSagaNotFound)
	}

	s.instances[instance.ID.String()] = instance

	return nil
}

// Load retrieves a saga instance by ID.
func (s *MemoryStore) Load(_ context.Context, id id.AggregateID) (*Instance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	instance, ok := s.instances[id.String()]
	if !ok {
		return nil, fmt.Errorf("saga %s: %w", id, ErrSagaNotFound)
	}

	return instance, nil
}

// LoadAllRunning returns all saga instances that are currently running or compensating.
func (s *MemoryStore) LoadAllRunning(_ context.Context) ([]*Instance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var running []*Instance

	for _, instance := range s.instances {
		if instance.Status == StatusRunning || instance.Status == StatusCompensating {
			running = append(running, instance)
		}
	}

	return running, nil
}
