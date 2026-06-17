package queries

import (
	"context"
	"encoding/json"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

const defaultPageSize = 20

type ListTodosQuery struct {
	*query.BasicQuery

	Status     *domain.TodoStatus `json:"status,omitempty"`
	Tags       []string           `json:"tags,omitempty"`
	Priority   *int               `json:"priority,omitempty"`
	Search     string             `json:"search,omitempty"`
	Pagination query.Pagination   `json:"pagination"`
}

func NewListTodosQuery() (*ListTodosQuery, error) {
	core, err := query.New(ListTodosQueryType)
	if err != nil {
		return nil, event.Newf(
			event.Infrastructure,
			"todo.queries.list_todos.1",
			"new list todos query: %v",
			err,
		)
	}

	return &ListTodosQuery{
		BasicQuery: core,
		Pagination: query.NewPagination(1, defaultPageSize),
	}, nil
}

type ListTodosResult struct {
	Todos []*GetTodoResult                      `json:"todos"`
	Page  query.PaginatedResult[*GetTodoResult] `json:"page"`
}

type ListTodosHandler struct{ readModel domain.TodoReadModel }

func NewListTodosHandler(readModel domain.TodoReadModel) *ListTodosHandler {
	return &ListTodosHandler{readModel: readModel}
}

func (h *ListTodosHandler) Handle(
	_ context.Context,
	q *ListTodosQuery,
) (*ListTodosResult, error) {
	filter := domain.TodoFilter{
		Status: q.Status, Tags: q.Tags,
		Priority: q.Priority, Search: q.Search,
		Limit:  int(q.Pagination.PageSize),
		Offset: q.Pagination.Offset(),
	}

	todos, err := h.readModel.List(filter)
	if err != nil {
		return nil, event.Newf(
			event.Infrastructure,
			"todo.queries.list_todos.2",
			"failed to list todos with filter %+v: %v",
			filter,
			err,
		)
	}

	results := make([]*GetTodoResult, len(todos))
	for i, todo := range todos {
		results[i] = FromDomain(todo)
	}

	return &ListTodosResult{
		Todos: results,
		Page:  query.NewPaginatedResult(results, uint(len(todos)), q.Pagination),
	}, nil
}

func (q *ListTodosQuery) MarshalJSON() ([]byte, error) {
	type Alias ListTodosQuery

	return json.Marshal(&struct {
		*Alias

		Type string `json:"type"`
	}{
		Type:  string(ListTodosQueryType),
		Alias: (*Alias)(q),
	})
}

var _ query.Query = (*ListTodosQuery)(nil)
