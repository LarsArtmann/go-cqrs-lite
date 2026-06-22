package queries

import (
	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

const (
	GetTodoQueryType    query.Type = "todo.get"
	ListTodosQueryType  query.Type = "todo.list"
	CountTodosQueryType query.Type = "todo.count"
)
