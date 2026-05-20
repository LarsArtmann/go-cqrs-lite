package queries

import (
	"encoding/json"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/query"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
)

type ListTodosQuery struct {
	*query.Core
	Status   *domain.TodoStatus `json:"status,omitempty"`
	Tags     []string           `json:"tags,omitempty"`
	Priority *int               `json:"priority,omitempty"`
	Search   string             `json:"search,omitempty"`
	Limit    int                `json:"limit"`
	Offset   int                `json:"offset"`
}

func NewListTodosQuery() (*ListTodosQuery, error) {
	core, err := query.New(ListTodosQueryType)
	if err != nil {
		return nil, fmt.Errorf("new list todos query: %w", err)
	}
	return &ListTodosQuery{Core: core, Limit: 20, Offset: 0}, nil
}

type ListTodosResult struct {
	Todos  []*GetTodoResult `json:"todos"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

type ListTodosHandler struct{ readModel domain.TodoReadModel }

func NewListTodosHandler(readModel domain.TodoReadModel) *ListTodosHandler {
	return &ListTodosHandler{readModel: readModel}
}

func (h *ListTodosHandler) Handle(q query.Query) (any, error) {
	listQuery, err := requireQueryType[*ListTodosQuery](q, "*ListTodosQuery")
	if err != nil {
		return nil, err
	}
	filter := domain.TodoFilter{
		Status: listQuery.Status, Tags: listQuery.Tags,
		Priority: listQuery.Priority, Search: listQuery.Search,
		Limit: listQuery.Limit, Offset: listQuery.Offset,
	}
	todos, err := h.readModel.List(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list todos with filter %+v: %w", filter, err)
	}
	results := make([]*GetTodoResult, len(todos))
	for i, todo := range todos {
		results[i] = FromDomain(todo)
	}
	return &ListTodosResult{Todos: results, Limit: listQuery.Limit, Offset: listQuery.Offset}, nil
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
