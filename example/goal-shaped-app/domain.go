// The Goal, in domain types. This file is the ENTIRE developer-owned data
// surface: plain Go structs. It imports nothing from go-cqrs-lite except the
// tier-1 core-domain bus types (command, query) — no engines, no schema, no
// table registration, no scan limits. Operators decide all of that in
// cqrs.yaml and can swap them per deployment without touching this file.
package main

import (
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// Task status values folded into [TaskView].
const (
	StatusOpen = "open"
	StatusDone = "done"
)

// TaskCreated opens a task.
type TaskCreated struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Priority int    `json:"priority"`
}

// TaskCompleted marks its task done. Every event carries the task ID —
// the read model derives each fold's key from it.
type TaskCompleted struct {
	ID string `json:"id"`
}

// TaskDeleted removes its task. The Evolution's Deleted-convention fold
// makes the view disappear — deletion is a domain event, never a mutation.
type TaskDeleted struct {
	ID string `json:"id"`
}

// TaskView is the read model. One Evolution derives it from the events
// above: Created/Deleted folds come from the naming convention, Completed
// is a single closure in app.go.
type TaskView struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
}

// CreateTaskCmd opens a task. The stream ID is the task identity.
type CreateTaskCmd struct {
	*command.BasicCommand

	Title    string
	Priority int
}

// CompleteTaskCmd marks an open task done.
type CompleteTaskCmd struct {
	*command.BasicCommand
}

// DeleteTaskCmd tombstones a task (ADR-0114: deletion is a domain event).
type DeleteTaskCmd struct {
	*command.BasicCommand
}

// GetTask asks for one task view by ID.
type GetTask struct {
	*query.BasicQuery

	ID string
}

// OpenTasks asks for every task whose status is still open.
type OpenTasks struct {
	*query.BasicQuery
}
