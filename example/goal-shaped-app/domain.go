package main

import (
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

//go:generate cqrs-gen --catalog .

// The Goal, in domain types. This file is the ENTIRE developer-owned data
// surface: plain Go structs. It imports nothing from go-cqrs-lite except the
// tier-1 core-domain bus types (command, query) and the tier-0 identity type
// (id): no engines, no schema, no table registration, no scan limits.
// Operators decide all of that in cqrs.yaml and can swap them per deployment
// without touching this file.
//
// Identifiers are branded end to end: the task identity is an id.StreamID in
// the events, the read model and the query alike, so another identity can
// never stand in for it. On the wire it stays a bare string (the catalog
// renders it as one), so branding costs nothing on the serialized surface.

// Task status values folded into [TaskView].
const (
	StatusOpen = "open"
	StatusDone = "done"

	// PriorityNormal and PriorityHigh are the story's demo priorities.
	PriorityNormal = 2
	PriorityHigh   = 5
)

// TaskCreated opens a task.
//
//cqrs:event task.created
type TaskCreated struct {
	ID       id.StreamID `json:"id"`
	Title    string      `json:"title"`
	Priority int         `json:"priority"`
}

// TaskUpdated changes a task, e.g. completing it with Status "done".
// The Updated suffix selects the convention's update fold — the payload is
// the full resulting row (convention folds mirror field-by-field, so the
// event carries Title/Priority too, sourced from the decider's state).
//
//cqrs:event task.updated
type TaskUpdated struct {
	ID       id.StreamID `json:"id"`
	Title    string      `json:"title"`
	Status   string      `json:"status"`
	Priority int         `json:"priority"`
}

// TaskDeleted removes its task. The Evolution's Deleted-convention fold
// makes the view disappear — deletion is a domain event, never a mutation.
//
//cqrs:event task.deleted
type TaskDeleted struct {
	ID id.StreamID `json:"id"`
}

// TaskView is the read model. One Evolution derives it from the events
// above: Created/Deleted folds come from the naming convention, Completed
// is a single closure in app.go.
type TaskView struct {
	ID       id.StreamID `json:"id"`
	Title    string      `json:"title"`
	Status   string      `json:"status"`
	Priority int         `json:"priority"`
}

// CreateTaskCmd opens a task. The stream ID is the task identity.
//
//cqrs:command task.create
type CreateTaskCmd struct {
	*command.BasicCommand

	Title    string
	Priority int
}

// CompleteTaskCmd marks an open task done.
//
//cqrs:command task.complete
type CompleteTaskCmd struct {
	*command.BasicCommand
}

// DeleteTaskCmd tombstones a task (ADR-0114: deletion is a domain event).
//
//cqrs:command task.delete
type DeleteTaskCmd struct {
	*command.BasicCommand
}

// GetTask asks for one task view by ID.
//
//cqrs:query task.get
type GetTask struct {
	*query.BasicQuery

	ID id.StreamID
}

// OpenTasks asks for every task whose status is still open.
//
//cqrs:query task.open
type OpenTasks struct {
	*query.BasicQuery
}
