package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/cockroachdb/pebble"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
)

var ErrBackupNotImplemented = errors.New("backup not implemented - use file-level backup")

type PebbleStore struct {
	PebbleMixin
}

func NewPebbleStore(dbPath string, logger *slog.Logger) (*PebbleStore, error) {
	if err := os.MkdirAll(dbPath, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create data directory %s: %w", dbPath, err)
	}
	opts := &pebble.Options{MaxOpenFiles: 1000}
	db, err := pebble.Open(dbPath, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open pebble db at %s: %w", dbPath, err)
	}
	return &PebbleStore{
		PebbleMixin: PebbleMixin{db: db, logger: logger, prefix: "todo:"},
	}, nil
}

func (s *PebbleStore) Close() error   { return s.db.Close() }
func (s *PebbleStore) DB() *pebble.DB { return s.db }

func (s *PebbleStore) key(id domain.TodoID) []byte { return []byte(s.prefix + id.String()) }

func (s *PebbleStore) Get(id domain.TodoID) (*domain.Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, closer, err := s.db.Get(s.key(id))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, fmt.Errorf("todo %s not found: %w", id.String(), domain.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get todo %s: %w", id.String(), err)
	}
	defer func() { _ = closer.Close() }()
	var todo domain.Todo
	if err := json.Unmarshal(value, &todo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal todo %s: %w", id, err)
	}
	return &todo, nil
}

func (s *PebbleStore) List(filter domain.TodoFilter) ([]*domain.Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var todos []*domain.Todo
	count := 0
	skipped := 0
	iter, err := newPrefixIter(s.db, s.prefix)
	if err != nil {
		return nil, fmt.Errorf("list todos: %w", err)
	}
	defer func() { _ = iter.Close() }()
	for iter.First(); iter.Valid(); iter.Next() {
		if skipped < filter.Offset {
			skipped++
			continue
		}
		if filter.Limit > 0 && count >= filter.Limit {
			break
		}
		var todo domain.Todo
		if !unmarshalFromIter(iter, s.logger, &todo) {
			continue
		}
		if !matchesFilter(&todo, filter) {
			continue
		}
		todos = append(todos, &todo)
		count++
	}
	return todos, handleIteratorError(iter, "list todos iterator")
}

func (s *PebbleStore) Put(todo *domain.Todo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.Marshal(todo)
	if err != nil {
		return fmt.Errorf("failed to marshal todo: %w", err)
	}
	if err := s.db.Set(s.key(todo.ID), data, pebble.Sync); err != nil {
		return fmt.Errorf("failed to put todo: %w", err)
	}
	return nil
}

func (s *PebbleStore) Delete(id domain.TodoID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.db.Delete(s.key(id), pebble.Sync); err != nil {
		return fmt.Errorf("failed to delete todo %s: %w", id.String(), err)
	}
	return nil
}

func (s *PebbleStore) Count(filter domain.TodoFilter) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	iter, err := newPrefixIter(s.db, s.prefix)
	if err != nil {
		return 0, fmt.Errorf("count todos: %w", err)
	}
	defer func() { _ = iter.Close() }()
	for iter.First(); iter.Valid(); iter.Next() {
		var todo domain.Todo
		if err := json.Unmarshal(iter.Value(), &todo); err != nil {
			continue
		}
		if matchesFilter(&todo, filter) {
			count++
		}
	}
	return count, nil
}
