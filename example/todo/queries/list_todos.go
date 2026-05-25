package queries

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/query"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
)

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
		return nil, fmt.Errorf("new list todos query: %w", err)
	}
	return &ListTodosQuery{
		BasicQuery:       core,
		Pagination: query.NewPagination(1, 20),
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

func (h *ListTodosHandler) Handle(ctx context.Context, q query.Query) (*ListTodosResult, error) {
	listQuery, err := requireQueryType[*ListTodosQuery](q, "*ListTodosQuery")
	if err != nil {
		return nil, err
	}
	filter := domain.TodoFilter{
		Status: listQuery.Status, Tags: listQuery.Tags,
		Priority: listQuery.Priority, Search: listQuery.Search,
		Limit:  int(listQuery.Pagination.PageSize),
		Offset: listQuery.Pagination.Offset(),
	}
	todos, err := h.readModel.List(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list todos with filter %+v: %w", filter, err)
	}
	results := make([]*GetTodoResult, len(todos))
	for i, todo := range todos {
		results[i] = FromDomain(todo)
	}
	return &ListTodosResult{
		Todos: results,
		Page:  query.NewPaginatedResult(results, uint(len(todos)), listQuery.Pagination),
	}, nil
}

func (q *ListTodosQuery) MarshalJSON() ([]byte, error) {
	type Alias ListTodosQuery
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  string(ListTodosQueryType),
		Alias: (*Alias)(q),
	})
}

var _ query.Query = (*ListTodosQuery)(nil)
