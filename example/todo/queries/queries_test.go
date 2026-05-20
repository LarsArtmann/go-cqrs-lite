package queries

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/query"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
)

func TestGetTodoHandler_Handle(t *testing.T) {
	t.Parallel()
	rm := &fakeReadModel{
		todos: map[domain.TodoID]*domain.Todo{
			testTodoID: {
				ID:          testTodoID,
				Title:       "Test Todo",
				Description: "Test Description",
				Status:      StatusPending,
				Priority:    3,
				Tags:        []string{"work"},
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Version:     1,
			},
		},
	}
	h := NewGetTodoHandler(rm)
	q, err := NewGetTodoQuery(testTodoID)
	if err != nil {
		t.Fatalf("NewGetTodoQuery: %v", err)
	}

	result, err := h.Handle(q)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	getResult, ok := result.(*GetTodoResult)
	if !ok {
		t.Fatalf("result type = %T, want *GetTodoResult", result)
	}
	if getResult.Title != "Test Todo" {
		t.Errorf("Title = %q, want %q", getResult.Title, "Test Todo")
	}
	if getResult.Description != "Test Description" {
		t.Errorf("Description = %q, want %q", getResult.Description, "Test Description")
	}
	if getResult.Status != StatusPending {
		t.Errorf("Status = %v, want %v", getResult.Status, StatusPending)
	}
}

func TestGetTodoHandler_Handle_NotFound(t *testing.T) {
	t.Parallel()
	rm := &fakeReadModel{todos: map[domain.TodoID]*domain.Todo{}}
	h := NewGetTodoHandler(rm)
	q, err := NewGetTodoQuery(testTodoID)
	if err != nil {
		t.Fatalf("NewGetTodoQuery: %v", err)
	}

	result, err := h.Handle(q)

	if result != nil {
		t.Errorf("result = %v, want nil", result)
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to get todo") {
		t.Errorf("error = %q, want containing %q", err.Error(), "failed to get todo")
	}
}

func TestGetTodoHandler_Handle_WrongQueryType(t *testing.T) {
	t.Parallel()
	rm := &fakeReadModel{}
	h := NewGetTodoHandler(rm)
	wrongQuery, _ := query.New(query.Type("wrong.type"))

	result, err := h.Handle(wrongQuery)

	if result != nil {
		t.Errorf("result = %v, want nil", result)
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid query type") {
		t.Errorf("error = %q, want containing %q", err.Error(), "invalid query type")
	}
}

func TestListTodosHandler_Handle(t *testing.T) {
	t.Parallel()
	priority := 3
	todo1 := domain.NewTodoID()
	todo2 := domain.NewTodoID()
	rm := &fakeReadModel{
		todoList: []*domain.Todo{
			{ID: todo1, Title: "Todo 1", Status: domain.StatusPending, Priority: 3},
			{ID: todo2, Title: "Todo 2", Status: domain.StatusCompleted, Priority: 3},
			{ID: domain.NewTodoID(), Title: "Todo 3", Status: domain.StatusInProgress, Priority: 5},
		},
	}
	h := NewListTodosHandler(rm)
	q, err := NewListTodosQuery()
	if err != nil {
		t.Fatalf("NewListTodosQuery: %v", err)
	}
	q.Priority = &priority
	q.Limit = 10

	result, err := h.Handle(q)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	listResult, ok := result.(*ListTodosResult)
	if !ok {
		t.Fatalf("result type = %T, want *ListTodosResult", result)
	}
	if len(listResult.Todos) != 2 {
		t.Errorf("Todos count = %d, want 2", len(listResult.Todos))
	}
	if listResult.Limit != 10 {
		t.Errorf("Limit = %d, want 10", listResult.Limit)
	}
}

func TestListTodosHandler_Handle_ErrorFromReadModel(t *testing.T) {
	t.Parallel()
	rm := &fakeReadModel{listErr: errors.New("read model unavailable")}
	h := NewListTodosHandler(rm)
	q, err := NewListTodosQuery()
	if err != nil {
		t.Fatalf("NewListTodosQuery: %v", err)
	}

	result, err := h.Handle(q)

	if result != nil {
		t.Errorf("result = %v, want nil", result)
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to list todos") {
		t.Errorf("error = %q, want containing %q", err.Error(), "failed to list todos")
	}
}

func TestCountTodosHandler_Handle(t *testing.T) {
	t.Parallel()
	rm := &fakeReadModel{
		todoList: []*domain.Todo{
			{ID: testTodoID, Title: "Todo 1", Status: domain.StatusPending},
			{ID: domain.NewTodoID(), Title: "Todo 2", Status: domain.StatusCompleted},
			{ID: domain.NewTodoID(), Title: "Todo 3", Status: domain.StatusCompleted},
		},
	}
	h := NewCountTodosHandler(rm)
	q, err := NewCountTodosQuery()
	if err != nil {
		t.Fatalf("NewCountTodosQuery: %v", err)
	}

	result, err := h.Handle(q)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	countResult, ok := result.(*CountTodosResult)
	if !ok {
		t.Fatalf("result type = %T, want *CountTodosResult", result)
	}
	if countResult.Count != 3 {
		t.Errorf("Count = %d, want 3", countResult.Count)
	}
}

func TestCountTodosHandler_Handle_ErrorFromReadModel(t *testing.T) {
	t.Parallel()
	rm := &fakeReadModel{countErr: errors.New("db unavailable")}
	h := NewCountTodosHandler(rm)
	q, err := NewCountTodosQuery()
	if err != nil {
		t.Fatalf("NewCountTodosQuery: %v", err)
	}

	result, err := h.Handle(q)

	if result != nil {
		t.Errorf("result = %v, want nil", result)
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to count todos") {
		t.Errorf("error = %q, want containing %q", err.Error(), "failed to count todos")
	}
}

func TestNewGetTodoQuery(t *testing.T) {
	t.Parallel()
	q, err := NewGetTodoQuery(testTodoID)
	if err != nil {
		t.Fatalf("NewGetTodoQuery: %v", err)
	}
	if q == nil {
		t.Fatal("query is nil")
	}
	if q.TodoID != testTodoID {
		t.Errorf("TodoID = %v, want %v", q.TodoID, testTodoID)
	}
}

func TestNewListTodosQuery(t *testing.T) {
	t.Parallel()
	q, err := NewListTodosQuery()
	if err != nil {
		t.Fatalf("NewListTodosQuery: %v", err)
	}
	if q == nil {
		t.Fatal("query is nil")
	}
	if q.Limit != 20 {
		t.Errorf("Limit = %d, want 20", q.Limit)
	}
	if q.Offset != 0 {
		t.Errorf("Offset = %d, want 0", q.Offset)
	}
}

func TestNewCountTodosQuery(t *testing.T) {
	t.Parallel()
	q, err := NewCountTodosQuery()
	if err != nil {
		t.Fatalf("NewCountTodosQuery: %v", err)
	}
	if q == nil {
		t.Fatal("query is nil")
	}
}

func TestGetTodoQuery_MarshalJSON(t *testing.T) {
	t.Parallel()
	q, err := NewGetTodoQuery(testTodoID)
	if err != nil {
		t.Fatalf("NewGetTodoQuery: %v", err)
	}

	data, err := q.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, `"type":"todo.get"`) {
		t.Errorf("JSON = %s, want containing type", s)
	}
	if !strings.Contains(s, testTodoID.String()) {
		t.Errorf("JSON = %s, want containing todo ID", s)
	}
}

var testTodoID = domain.NewTodoID()

type fakeReadModel struct {
	domain.TodoReadModel
	todos    map[domain.TodoID]*domain.Todo
	todoList []*domain.Todo
	listErr  error
	countErr error
}

func (f *fakeReadModel) Get(id domain.TodoID) (*domain.Todo, error) {
	if f.todos == nil {
		return nil, domain.ErrNotFound
	}
	todo, ok := f.todos[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return todo, nil
}

func (f *fakeReadModel) List(filter domain.TodoFilter) ([]*domain.Todo, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	results := make([]*domain.Todo, 0)
	for _, todo := range f.todoList {
		if filter.Priority != nil && todo.Priority != *filter.Priority {
			continue
		}
		if filter.Status != nil && todo.Status != *filter.Status {
			continue
		}
		results = append(results, todo)
	}
	return results, nil
}

func (f *fakeReadModel) Count(filter domain.TodoFilter) (int, error) {
	if f.countErr != nil {
		return 0, f.countErr
	}
	todos, err := f.List(filter)
	if err != nil {
		return 0, err
	}
	return len(todos), nil
}

var _ domain.TodoReadModel = (*fakeReadModel)(nil)

var StatusPending = domain.StatusPending
