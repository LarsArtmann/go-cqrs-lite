package queries

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/query"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
)

type CountTodosQuery struct {
	*query.Core
	Status   *domain.TodoStatus `json:"status,omitempty"`
	Tags     []string           `json:"tags,omitempty"`
	Priority *int               `json:"priority,omitempty"`
	Search   string             `json:"search,omitempty"`
}

func NewCountTodosQuery() (*CountTodosQuery, error) {
	core, err := query.New(CountTodosQueryType)
	if err != nil {
		return nil, fmt.Errorf("new count todos query: %w", err)
	}
	return &CountTodosQuery{Core: core}, nil
}

type CountTodosResult struct {
	Count int `json:"count"`
}

type CountTodosHandler struct{ readModel domain.TodoReadModel }

func NewCountTodosHandler(readModel domain.TodoReadModel) *CountTodosHandler {
	return &CountTodosHandler{readModel: readModel}
}

func (h *CountTodosHandler) Handle(q query.Query) (any, error) {
	countQuery, err := requireQueryType[*CountTodosQuery](q, "*CountTodosQuery")
	if err != nil {
		return nil, err
	}
	filter := domain.TodoFilter{
		Status: countQuery.Status, Tags: countQuery.Tags,
		Priority: countQuery.Priority, Search: countQuery.Search,
	}
	count, err := h.readModel.Count(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count todos with filter %+v: %w", filter, err)
	}
	return &CountTodosResult{Count: count}, nil
}

func (q *CountTodosQuery) MarshalJSON() ([]byte, error) {
	return marshalQueryJSON(q, CountTodosQueryType)
}

var _ query.Query = (*CountTodosQuery)(nil)
