package queries

import (
	"encoding/json"
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

func marshalQueryJSON(q any, qType query.Type) ([]byte, error) {
	type result struct {
		Type string `json:"type"`
	}
	data, err := json.Marshal(q)
	if err != nil {
		return nil, fmt.Errorf("marshal query %s: %w", qType, err)
	}
	var base map[string]any
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, fmt.Errorf("unmarshal query %s: %w", qType, err)
	}
	base["type"] = string(qType)
	return json.Marshal(base)
}
