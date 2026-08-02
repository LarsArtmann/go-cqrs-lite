package main

import (
	"context"
	"time"

	gomust "github.com/larsartmann/go-must"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// ──────────────────────────────────────────────────────────────────────────
// Command and Query types — these embed command.BasicCommand / query.BasicQuery
// to satisfy the dispatcher interfaces, and carry the domain-specific payload.
//
// The decider functions in decider.go use plain structs (CreateTask, etc.)
// that know nothing about dispatching. These types bridge the transport layer
// to the decider layer.
// ──────────────────────────────────────────────────────────────────────────

const (
	cmdCreateTask   = command.Type("task.create")
	cmdAssignTask   = command.Type("task.assign")
	cmdStartTask    = command.Type("task.start")
	cmdCompleteTask = command.Type("task.complete")
	cmdArchiveTask  = command.Type("task.archive")
	cmdDeleteTask   = command.Type("task.delete")
	cmdUpdateTitle  = command.Type("task.update_title")
	cmdChangePrio   = command.Type("task.change_priority")
	cmdSetDueDate   = command.Type("task.set_due_date")
	cmdAddBlocker   = command.Type("task.add_blocker")
)

// CreateTaskCmd carries the payload for creating a new task.
//
//cqrs-lint:ignore(B004) library code or intentional pattern
type CreateTaskCmd struct {
	*command.BasicCommand

	Title       string
	Description string
	Priority    Priority
}

type AssignTaskCmd struct {
	*command.BasicCommand

	AssigneeID string
}

type (
	StartTaskCmd    struct{ *command.BasicCommand }
	CompleteTaskCmd struct{ *command.BasicCommand }
	ArchiveTaskCmd  struct{ *command.BasicCommand }
	DeleteTaskCmd   struct{ *command.BasicCommand }
)

type UpdateTitleCmd struct {
	*command.BasicCommand

	Title string
}

type ChangePriorityCmd struct {
	*command.BasicCommand

	Priority Priority
}

type SetDueDateCmd struct {
	*command.BasicCommand

	DueDate *time.Time
}

type AddBlockerCmd struct {
	*command.BasicCommand

	DependencyID id.StreamID
}

// GetTaskQuery is the query to fetch a single task by ID.
//
//cqrs-lint:ignore(E007) library code or intentional pattern
type GetTaskQuery struct {
	*query.BasicQuery
}

type GetTaskResult struct {
	Task *TaskView `json:"task"`
}

// ListTasksQuery filters tasks by status.
//
//cqrs-lint:ignore(E007) library code or intentional pattern
type ListTasksQuery struct {
	*query.BasicQuery

	StatusFilter string
}

type ListTasksResult struct {
	Tasks []*TaskView `json:"tasks"`
}

// ──────────────────────────────────────────────────────────────────────────
// registerHandlers wires command and query handlers to the dispatchers.
// ──────────────────────────────────────────────────────────────────────────

func registerHandlers(s *Server) {
	gomust.Check(command.RegisterTyped(s.CmdDisp, cmdCreateTask,
		func(ctx context.Context, cmd CreateTaskCmd) error {
			return s.Repo.Execute(ctx, cmd.StreamID(), streamType,
				Create(CreateTask{
					ID: cmd.StreamID(), Title: cmd.Title,
					Description: cmd.Description, Priority: cmd.Priority,
				}))
		}))

	gomust.Check(command.RegisterTyped(s.CmdDisp, cmdAssignTask,
		func(ctx context.Context, cmd AssignTaskCmd) error {
			return s.Repo.Execute(ctx, cmd.StreamID(), streamType,
				Assign(AssignTask{ID: cmd.StreamID(), AssigneeID: cmd.AssigneeID}))
		}))

	gomust.Check(command.RegisterTyped(s.CmdDisp, cmdStartTask,
		func(ctx context.Context, cmd StartTaskCmd) error {
			return s.Repo.Execute(ctx, cmd.StreamID(), streamType,
				Start(StartTask{ID: cmd.StreamID()}))
		}))

	gomust.Check(command.RegisterTyped(s.CmdDisp, cmdCompleteTask,
		func(ctx context.Context, cmd CompleteTaskCmd) error {
			return s.Repo.Execute(ctx, cmd.StreamID(), streamType,
				Complete(CompleteTask{ID: cmd.StreamID()}))
		}))

	gomust.Check(command.RegisterTyped(s.CmdDisp, cmdArchiveTask,
		func(ctx context.Context, cmd ArchiveTaskCmd) error {
			return s.Repo.Execute(ctx, cmd.StreamID(), streamType,
				Archive(ArchiveTask{ID: cmd.StreamID()}))
		}))

	gomust.Check(command.RegisterTyped(s.CmdDisp, cmdDeleteTask,
		func(ctx context.Context, cmd DeleteTaskCmd) error {
			return s.Repo.Execute(ctx, cmd.StreamID(), streamType,
				Delete(DeleteTask{ID: cmd.StreamID()}))
		}))

	gomust.Check(command.RegisterTyped(s.CmdDisp, cmdUpdateTitle,
		func(ctx context.Context, cmd UpdateTitleCmd) error {
			return s.Repo.Execute(ctx, cmd.StreamID(), streamType,
				UpdateTaskTitle(UpdateTitle{ID: cmd.StreamID(), Title: cmd.Title}))
		}))

	gomust.Check(command.RegisterTyped(s.CmdDisp, cmdChangePrio,
		func(ctx context.Context, cmd ChangePriorityCmd) error {
			return s.Repo.Execute(ctx, cmd.StreamID(), streamType,
				ChangeTaskPriority(ChangePriority{ID: cmd.StreamID(), Priority: cmd.Priority}))
		}))

	gomust.Check(command.RegisterTyped(s.CmdDisp, cmdSetDueDate,
		func(ctx context.Context, cmd SetDueDateCmd) error {
			return s.Repo.Execute(ctx, cmd.StreamID(), streamType,
				SetTaskDueDate(SetDueDate{ID: cmd.StreamID(), DueDate: cmd.DueDate}))
		}))

	gomust.Check(command.RegisterTyped(s.CmdDisp, cmdAddBlocker,
		func(ctx context.Context, cmd AddBlockerCmd) error {
			return s.Repo.Execute(ctx, cmd.StreamID(), streamType,
				AddBlocker(BlockBy{ID: cmd.StreamID(), DependencyID: cmd.DependencyID}))
		}))
}
