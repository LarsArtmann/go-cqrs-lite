package queries

import (
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"context"
	"encoding/json"

	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

type CountTodosQuery struct {
	*query.BasicQuery

	Status   *domain.TodoStatus `json:"status,omitempty"`
	Tags     []string           `json:"tags,omitempty"`
	Priority *int               `json:"priority,omitempty"`
	Search   string             `json:"search,omitempty"`
}

func NewCountTodosQuery() (*CountTodosQuery, error) {
	core, err := query.New(CountTodosQueryType)
	if err != nil {
		return nil, event.Newf(event.Infrastructure, "todo.queries.count_todos.1", "new count todos query: %v", err)
	}

	return &CountTodosQuery{BasicQuery: core}, nil
}

type CountTodosResult struct {
	Count int `json:"count"`
}

type CountTodosHandler struct{ readModel domain.TodoReadModel }

func NewCountTodosHandler(readModel domain.TodoReadModel) *CountTodosHandler {
	return &CountTodosHandler{readModel: readModel}
}

func (h *CountTodosHandler) Handle(
	_ context.Context,
	q *CountTodosQuery,
) (*CountTodosResult, error) {
	filter := domain.TodoFilter{
		Status: q.Status, Tags: q.Tags,
		Priority: q.Priority, Search: q.Search,
	}

	count, err := h.readModel.Count(filter)
	if err != nil {
		return nil, event.Newf(event.Infrastructure, "todo.queries.count_todos.2", "failed to count todos with filter %+v: %v", filter, err)
	}

	return &CountTodosResult{Count: count}, nil
}

func (q *CountTodosQuery) MarshalJSON() ([]byte, error) {
	type Alias CountTodosQuery

	return json.Marshal(&struct {
		*Alias

		Type string `json:"type"`
	}{
		Type:  string(CountTodosQueryType),
		Alias: (*Alias)(q),
	})
}

var _ query.Query = (*CountTodosQuery)(nil)
