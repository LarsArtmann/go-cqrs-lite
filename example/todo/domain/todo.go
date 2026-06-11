package domain

import (
	"errors"
	"time"
)

var (
	ErrEmptyTitle     = errors.New("todo title cannot be empty")
	ErrInvalidStatus  = errors.New("invalid todo status")
	ErrNotFound       = errors.New("todo not found")
	ErrConcurrentEdit = errors.New("concurrent edit detected")
)

type TodoStatus string

const (
	StatusPending    TodoStatus = "pending"
	StatusInProgress TodoStatus = "in_progress"
	StatusCompleted  TodoStatus = "completed"
	StatusArchived   TodoStatus = "archived"
)

func (s TodoStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusInProgress, StatusCompleted, StatusArchived:
		return true
	}

	return false
}

type Title string

func (t Title) String() string { return string(t) }

func (t Title) IsZero() bool { return t == "" }

type Description string

func (d Description) String() string { return string(d) }

func (d Description) IsZero() bool { return d == "" }

type Priority int

func (p Priority) Int() int { return int(p) }

func (p Priority) IsZero() bool { return p == 0 }

type Todo struct {
	ID          TodoID     `json:"id"`
	Title       Title      `json:"title"`
	Description Description `json:"description"`
	Status      TodoStatus `json:"status"`
	Priority    Priority   `json:"priority"`
	Tags        []string   `json:"tags"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	Version     int64      `json:"version"`
}

func (t *Todo) Clone() *Todo {
	cloned := &Todo{
		ID: t.ID, Title: t.Title, Description: t.Description,
		Status: t.Status, Priority: t.Priority,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt, Version: t.Version,
	}
	if t.CompletedAt != nil {
		completed := *t.CompletedAt
		cloned.CompletedAt = &completed
	}

	if len(t.Tags) > 0 {
		cloned.Tags = make([]string, len(t.Tags))
		copy(cloned.Tags, t.Tags)
	} else {
		cloned.Tags = make([]string, 0)
	}

	return cloned
}

type TodoFilter struct {
	Status   *TodoStatus
	Tags     []string
	Priority *int
	Search   string
	Limit    int
	Offset   int
}

type TodoReadModel interface {
	Get(id TodoID) (*Todo, error)
	List(filter TodoFilter) ([]*Todo, error)
	Put(todo *Todo) error
	Delete(id TodoID) error
	Count(filter TodoFilter) (int, error)
}
