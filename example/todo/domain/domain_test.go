package domain

import (
	"testing"
	"time"
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
			if got := tt.status.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
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

	if cloned.ID != todo.ID {
		t.Errorf("ID = %v, want %v", cloned.ID, todo.ID)
	}
	if cloned.Title != todo.Title {
		t.Errorf("Title = %v, want %v", cloned.Title, todo.Title)
	}
	if cloned.Description != todo.Description {
		t.Errorf("Description = %v, want %v", cloned.Description, todo.Description)
	}
	if cloned.Status != todo.Status {
		t.Errorf("Status = %v, want %v", cloned.Status, todo.Status)
	}
	if cloned.Priority != todo.Priority {
		t.Errorf("Priority = %v, want %v", cloned.Priority, todo.Priority)
	}
	if cloned.Version != todo.Version {
		t.Errorf("Version = %v, want %v", cloned.Version, todo.Version)
	}
	if cloned.CompletedAt == nil {
		t.Fatal("CompletedAt is nil, want non-nil")
	}
	if *cloned.CompletedAt != *todo.CompletedAt {
		t.Errorf("CompletedAt = %v, want %v", *cloned.CompletedAt, *todo.CompletedAt)
	}
	if len(cloned.Tags) != 2 {
		t.Fatalf("Tags length = %d, want 2", len(cloned.Tags))
	}
	if cloned.Tags[0] != todo.Tags[0] || cloned.Tags[1] != todo.Tags[1] {
		t.Errorf("Tags = %v, want %v", cloned.Tags, todo.Tags)
	}
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

	if cloned.Tags == nil {
		t.Fatal("Tags is nil, want non-nil")
	}
	if len(cloned.Tags) != 0 {
		t.Errorf("Tags length = %d, want 0", len(cloned.Tags))
	}
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

	if cloned.CompletedAt != nil {
		t.Errorf("CompletedAt = %v, want nil", cloned.CompletedAt)
	}
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

	if todo.Title != "Original" {
		t.Errorf("Title = %q, want %q", todo.Title, "Original")
	}
	if todo.Tags[0] != "original" {
		t.Errorf("Tags[0] = %q, want %q", todo.Tags[0], "original")
	}
	if todo.Status != StatusPending {
		t.Errorf("Status = %v, want %v", todo.Status, StatusPending)
	}
}

func TestNewTodoID(t *testing.T) {
	t.Parallel()
	id1 := NewTodoID()
	id2 := NewTodoID()

	if id1 == id2 {
		t.Error("two generated IDs should differ")
	}
	if id1.String() == "" {
		t.Error("ID string should not be empty")
	}
}

func TestParseTodoID(t *testing.T) {
	t.Parallel()
	id := NewTodoID()
	s := id.String()

	parsed, err := ParseTodoID(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed != id {
		t.Errorf("parsed = %v, want %v", parsed, id)
	}
}

func TestParseTodoID_Invalid(t *testing.T) {
	t.Parallel()
	_, err := ParseTodoID("not-a-valid-id")
	if err == nil {
		t.Error("expected error for invalid ID, got nil")
	}
}
