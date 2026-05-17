package storage

import (
	"fmt"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
)

type MemoryStore struct {
	mu    sync.RWMutex
	todos map[string]*domain.Todo
}

func NewMemoryStore() (*MemoryStore, func()) {
	return &MemoryStore{todos: make(map[string]*domain.Todo)}, func() {}
}

func (s *MemoryStore) Get(id domain.TodoID) (*domain.Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	todo, ok := s.todos[id.String()]
	if !ok {
		return nil, fmt.Errorf("todo %s not found: %w", id.String(), domain.ErrNotFound)
	}
	return todo.Clone(), nil
}

func (s *MemoryStore) List(filter domain.TodoFilter) ([]*domain.Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var todos []*domain.Todo
	count := 0
	skipped := 0
	for _, todo := range s.todos {
		if !matchesFilter(todo, filter) {
			continue
		}
		if skipped < filter.Offset {
			skipped++
			continue
		}
		if filter.Limit > 0 && count >= filter.Limit {
			break
		}
		todos = append(todos, todo.Clone())
		count++
	}
	return todos, nil
}

func (s *MemoryStore) Put(todo *domain.Todo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.todos[todo.ID.String()] = todo.Clone()
	return nil
}

func (s *MemoryStore) Delete(id domain.TodoID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.todos[id.String()]; !exists {
		return fmt.Errorf("todo %s not found: %w", id.String(), domain.ErrNotFound)
	}
	delete(s.todos, id.String())
	return nil
}

func (s *MemoryStore) Count(filter domain.TodoFilter) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, todo := range s.todos {
		if matchesFilter(todo, filter) {
			count++
		}
	}
	return count, nil
}
