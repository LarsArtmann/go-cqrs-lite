package queries

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/query"
)

const (
	GetTodoQueryType    query.Type = "todo.get"
	ListTodosQueryType  query.Type = "todo.list"
	CountTodosQueryType query.Type = "todo.count"
)

func requireQueryType[T any](q query.Query, expected string) (T, error) {
	var zero T
	if q == nil {
		return zero, fmt.Errorf("invalid query type: expected %s, got nil", expected)
	}

	typed, ok := q.(T)
	if !ok {
		return zero, fmt.Errorf("invalid query type: expected %s, got %T", expected, q)
	}

	return typed, nil
}
