package queries

import (
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/query"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)

	result, err := h.Handle(q)

	require.NoError(t, err)
	getResult, ok := result.(*GetTodoResult)
	require.True(t, ok)
	assert.Equal(t, "Test Todo", getResult.Title)
	assert.Equal(t, "Test Description", getResult.Description)
	assert.Equal(t, StatusPending, getResult.Status)
}

func TestGetTodoHandler_Handle_NotFound(t *testing.T) {
	t.Parallel()
	rm := &fakeReadModel{todos: map[domain.TodoID]*domain.Todo{}}
	h := NewGetTodoHandler(rm)
	q, err := NewGetTodoQuery(testTodoID)
	require.NoError(t, err)

	result, err := h.Handle(q)

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get todo")
}

func TestGetTodoHandler_Handle_WrongQueryType(t *testing.T) {
	t.Parallel()
	rm := &fakeReadModel{}
	h := NewGetTodoHandler(rm)
	wrongQuery, _ := query.New(query.Type("wrong.type"))

	result, err := h.Handle(wrongQuery)

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid query type")
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
	require.NoError(t, err)
	q.Priority = &priority
	q.Limit = 10

	result, err := h.Handle(q)

	require.NoError(t, err)
	listResult, ok := result.(*ListTodosResult)
	require.True(t, ok)
	assert.Len(t, listResult.Todos, 2)
	assert.Equal(t, 10, listResult.Limit)
}

func TestListTodosHandler_Handle_ErrorFromReadModel(t *testing.T) {
	t.Parallel()
	rm := &fakeReadModel{listErr: errors.New("read model unavailable")}
	h := NewListTodosHandler(rm)
	q, err := NewListTodosQuery()
	require.NoError(t, err)

	result, err := h.Handle(q)

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list todos")
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
	require.NoError(t, err)

	result, err := h.Handle(q)

	require.NoError(t, err)
	countResult, ok := result.(*CountTodosResult)
	require.True(t, ok)
	assert.Equal(t, 3, countResult.Count)
}

func TestCountTodosHandler_Handle_ErrorFromReadModel(t *testing.T) {
	t.Parallel()
	rm := &fakeReadModel{countErr: errors.New("db unavailable")}
	h := NewCountTodosHandler(rm)
	q, err := NewCountTodosQuery()
	require.NoError(t, err)

	result, err := h.Handle(q)

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to count todos")
}

func TestNewGetTodoQuery(t *testing.T) {
	t.Parallel()
	q, err := NewGetTodoQuery(testTodoID)
	require.NoError(t, err)
	require.NotNil(t, q)
	assert.Equal(t, testTodoID, q.TodoID)
}

func TestNewListTodosQuery(t *testing.T) {
	t.Parallel()
	q, err := NewListTodosQuery()
	require.NoError(t, err)
	require.NotNil(t, q)
	assert.Equal(t, 20, q.Limit)
	assert.Equal(t, 0, q.Offset)
}

func TestNewCountTodosQuery(t *testing.T) {
	t.Parallel()
	q, err := NewCountTodosQuery()
	require.NoError(t, err)
	require.NotNil(t, q)
}

func TestGetTodoQuery_MarshalJSON(t *testing.T) {
	t.Parallel()
	q, err := NewGetTodoQuery(testTodoID)
	require.NoError(t, err)

	data, err := q.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(data), `"type":"todo.get"`)
	assert.Contains(t, string(data), testTodoID.String())
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
