package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTodoStatus_IsValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		status   TodoStatus
		expected bool
	}{
		{StatusPending, true},
		{StatusInProgress, true},
		{StatusCompleted, true},
		{StatusArchived, true},
		{TodoStatus("invalid"), false},
		{TodoStatus(""), false},
		{TodoStatus("PENDING"), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.status.IsValid())
		})
	}
}

func TestTodo_Clone(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	completedAt := time.Now().UTC()
	todo := &Todo{
		ID:          NewTodoID(),
		Title:       "Test Todo",
		Description: "Test Description",
		Status:      StatusCompleted,
		Priority:    5,
		Tags:        []string{"work", "urgent"},
		CreatedAt:   now,
		UpdatedAt:   now,
		CompletedAt: &completedAt,
		Version:     3,
	}

	cloned := todo.Clone()

	assert.Equal(t, todo.ID, cloned.ID)
	assert.Equal(t, todo.Title, cloned.Title)
	assert.Equal(t, todo.Description, cloned.Description)
	assert.Equal(t, todo.Status, cloned.Status)
	assert.Equal(t, todo.Priority, cloned.Priority)
	assert.Equal(t, todo.Version, cloned.Version)
	require.NotNil(t, cloned.CompletedAt)
	assert.Equal(t, *todo.CompletedAt, *cloned.CompletedAt)
	require.Len(t, cloned.Tags, 2)
	assert.Equal(t, todo.Tags, cloned.Tags)
}

func TestTodo_Clone_EmptyTags(t *testing.T) {
	t.Parallel()
	todo := &Todo{
		ID:        NewTodoID(),
		Title:     "No Tags",
		Tags:      []string{},
		Status:    StatusPending,
		Version:   1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	cloned := todo.Clone()

	require.NotNil(t, cloned.Tags)
	assert.Empty(t, cloned.Tags)
}

func TestTodo_Clone_NilCompletedAt(t *testing.T) {
	t.Parallel()
	todo := &Todo{
		ID:          NewTodoID(),
		Title:       "Not Completed",
		Status:      StatusPending,
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CompletedAt: nil,
	}

	cloned := todo.Clone()

	assert.Nil(t, cloned.CompletedAt)
}

func TestTodo_Clone_Immutability(t *testing.T) {
	t.Parallel()
	todo := &Todo{
		ID:        NewTodoID(),
		Title:     "Original",
		Tags:      []string{"original"},
		Status:    StatusPending,
		Version:   1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	cloned := todo.Clone()
	cloned.Title = "Modified"
	cloned.Tags[0] = "modified"
	cloned.Status = StatusCompleted

	assert.Equal(t, "Original", todo.Title)
	assert.Equal(t, "original", todo.Tags[0])
	assert.Equal(t, StatusPending, todo.Status)
}

func TestNewTodoID(t *testing.T) {
	t.Parallel()
	id1 := NewTodoID()
	id2 := NewTodoID()

	assert.NotEqual(t, id1, id2)
	assert.NotEmpty(t, id1.String())
}

func TestParseTodoID(t *testing.T) {
	t.Parallel()
	id := NewTodoID()
	s := id.String()

	parsed, err := ParseTodoID(s)
	require.NoError(t, err)
	assert.Equal(t, id, parsed)
}

func TestParseTodoID_Invalid(t *testing.T) {
	t.Parallel()
	_, err := ParseTodoID("not-a-valid-id")
	assert.Error(t, err)
}
