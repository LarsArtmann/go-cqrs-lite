package queries

import (
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/query"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
)

type GetTodoQuery struct {
	*query.Core
	TodoID domain.TodoID `json:"todo_id"`
}

func NewGetTodoQuery(todoID domain.TodoID) (*GetTodoQuery, error) {
	core, err := query.New(GetTodoQueryType)
	if err != nil {
		return nil, fmt.Errorf("new get todo query for todo %s: %w", todoID, err)
	}
	return &GetTodoQuery{Core: core, TodoID: todoID}, nil
}

type GetTodoResult struct {
	ID          domain.TodoID     `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      domain.TodoStatus `json:"status"`
	Priority    int               `json:"priority"`
	Tags        []string          `json:"tags"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	Version     int64             `json:"version"`
}

func FromDomain(t *domain.Todo) *GetTodoResult {
	return &GetTodoResult{
		ID: t.ID, Title: t.Title, Description: t.Description,
		Status: t.Status, Priority: t.Priority, Tags: t.Tags,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
		CompletedAt: t.CompletedAt, Version: t.Version,
	}
}

type GetTodoHandler struct{ readModel domain.TodoReadModel }

func NewGetTodoHandler(readModel domain.TodoReadModel) *GetTodoHandler {
	return &GetTodoHandler{readModel: readModel}
}

func (h *GetTodoHandler) Handle(q query.Query) (any, error) {
	getQuery, err := requireQueryType[*GetTodoQuery](q, "*GetTodoQuery")
	if err != nil {
		return nil, err
	}
	todo, err := h.readModel.Get(getQuery.TodoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get todo %s: %w", getQuery.TodoID, err)
	}
	return FromDomain(todo), nil
}

func (q *GetTodoQuery) MarshalJSON() ([]byte, error) {
	return marshalQueryJSON(q, GetTodoQueryType)
}

var _ query.Query = (*GetTodoQuery)(nil)
