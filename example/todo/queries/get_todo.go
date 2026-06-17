package queries

import (
	"context"
	"encoding/json"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

type GetTodoQuery struct {
	*query.BasicQuery

	TodoID domain.TodoID `json:"todoId"`
}

func NewGetTodoQuery(todoID domain.TodoID) (*GetTodoQuery, error) {
	core, err := query.New(GetTodoQueryType)
	if err != nil {
		return nil, event.Newf(
			event.Infrastructure,
			"todo.queries.get_todo.1",
			"new get todo query for todo %s: %v",
			todoID,
			err,
		)
	}

	return &GetTodoQuery{BasicQuery: core, TodoID: todoID}, nil
}

type GetTodoResult struct {
	ID          domain.TodoID     `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      domain.TodoStatus `json:"status"`
	Priority    int               `json:"priority"`
	Tags        []string          `json:"tags"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	CompletedAt *time.Time        `json:"completedAt,omitempty"`
	Version     int64             `json:"version"`
}

func FromDomain(t *domain.Todo) *GetTodoResult {
	return &GetTodoResult{
		ID: t.ID, Title: string(t.Title), Description: string(t.Description),
		Status: t.Status, Priority: int(t.Priority), Tags: t.Tags,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
		CompletedAt: t.CompletedAt, Version: t.Version,
	}
}

type GetTodoHandler struct{ readModel domain.TodoReadModel }

func NewGetTodoHandler(readModel domain.TodoReadModel) *GetTodoHandler {
	return &GetTodoHandler{readModel: readModel}
}

func (h *GetTodoHandler) Handle(_ context.Context, q *GetTodoQuery) (*GetTodoResult, error) {
	todo, err := h.readModel.Get(q.TodoID)
	if err != nil {
		return nil, event.Newf(
			event.Infrastructure,
			"todo.queries.get_todo.2",
			"failed to get todo %s: %v",
			q.TodoID,
			err,
		)
	}

	return FromDomain(todo), nil
}

func (q *GetTodoQuery) MarshalJSON() ([]byte, error) {
	type Alias GetTodoQuery

	return json.Marshal(&struct {
		*Alias

		Type string `json:"type"`
	}{
		Type:  string(GetTodoQueryType),
		Alias: (*Alias)(q),
	})
}

var _ query.Query = (*GetTodoQuery)(nil)
