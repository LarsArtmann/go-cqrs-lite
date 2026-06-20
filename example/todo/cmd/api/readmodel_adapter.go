package main

import (
	"context"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
	"github.com/larsartmann/go-cqrs-lite/kv/v2"
)

// readModelAdapter bridges kv.TypedStore[Todo, TodoID] to domain.TodoReadModel.
//
// This is the consumer-side glue: the library provides a typed key-value store,
// and the consumer wraps it with domain-specific query logic (filtering by
// status, tags, priority, full-text search). List and Count scan + filter
// in memory; for large datasets, a consumer would add secondary indexes.
type readModelAdapter struct {
	store *kv.TypedStore[domain.Todo, domain.TodoID]
}

var _ domain.TodoReadModel = (*readModelAdapter)(nil)

func newReadModelAdapter(store *kv.TypedStore[domain.Todo, domain.TodoID]) *readModelAdapter {
	return &readModelAdapter{store: store}
}

func (a *readModelAdapter) Get(id domain.TodoID) (*domain.Todo, error) {
	return a.store.Get(context.Background(), id)
}

func (a *readModelAdapter) Put(todo *domain.Todo) error {
	return a.store.Set(context.Background(), todo.ID, todo)
}

func (a *readModelAdapter) Delete(id domain.TodoID) error {
	return a.store.Delete(context.Background(), id)
}

func (a *readModelAdapter) List(filter domain.TodoFilter) ([]*domain.Todo, error) {
	all, err := a.store.Scan(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Todo, 0, len(all))

	for _, todo := range all {
		if todoMatchesFilter(todo, filter) {
			result = append(result, todo)
		}
	}

	if filter.Offset > 0 && filter.Offset < len(result) {
		result = result[filter.Offset:]
	}

	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[:filter.Limit]
	}

	return result, nil
}

func (a *readModelAdapter) Count(filter domain.TodoFilter) (int, error) {
	all, err := a.store.Scan(context.Background(), nil)
	if err != nil {
		return 0, err
	}

	count := 0

	for _, todo := range all {
		if todoMatchesFilter(todo, filter) {
			count++
		}
	}

	return count, nil
}

func todoMatchesFilter(todo *domain.Todo, filter domain.TodoFilter) bool {
	if filter.Status != nil && todo.Status != *filter.Status {
		return false
	}

	if filter.Priority != nil && todo.Priority != domain.Priority(*filter.Priority) {
		return false
	}

	if len(filter.Tags) > 0 {
		tagSet := make(map[string]struct{}, len(todo.Tags))

		for _, tag := range todo.Tags {
			tagSet[tag] = struct{}{}
		}

		for _, tag := range filter.Tags {
			if _, ok := tagSet[tag]; !ok {
				return false
			}
		}
	}

	if filter.Search != "" {
		searchLower := strings.ToLower(filter.Search)
		if !strings.Contains(strings.ToLower(string(todo.Title)), searchLower) &&
			!strings.Contains(strings.ToLower(string(todo.Description)), searchLower) {
			return false
		}
	}

	return true
}
