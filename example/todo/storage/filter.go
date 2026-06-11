package storage

import (
	"strings"

	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
)

func matchesFilter(todo *domain.Todo, filter domain.TodoFilter) bool {
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
