package main

import (
	"context"
	"fmt"
	"time"

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

	DependencyID id.AggregateID
}

// Query types.
const (
	qryGetTask = query.Type("task.get")
	qryListAll = query.Type("task.list")
)

type GetTaskQuery struct {
	*query.BasicQuery
}

type GetTaskResult struct {
	Task *TaskView `json:"task"`
}

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
	must := func(err error) {
		if err != nil {
			panic(fmt.Sprintf("register handler: %v", err))
		}
	}

	must(command.RegisterTyped(s.CmdDisp, cmdCreateTask,
		func(ctx context.Context, cmd CreateTaskCmd) error {
			return s.Repo.Execute(ctx, cmd.AggregateID(), aggregateType,
				Create(CreateTask{
					ID: cmd.AggregateID(), Title: cmd.Title,
					Description: cmd.Description, Priority: cmd.Priority,
				}))
		}))

	must(command.RegisterTyped(s.CmdDisp, cmdAssignTask,
		func(ctx context.Context, cmd AssignTaskCmd) error {
			return s.Repo.Execute(ctx, cmd.AggregateID(), aggregateType,
				Assign(AssignTask{ID: cmd.AggregateID(), AssigneeID: cmd.AssigneeID}))
		}))

	must(command.RegisterTyped(s.CmdDisp, cmdStartTask,
		func(ctx context.Context, cmd StartTaskCmd) error {
			return s.Repo.Execute(ctx, cmd.AggregateID(), aggregateType,
				Start(StartTask{ID: cmd.AggregateID()}))
		}))

	must(command.RegisterTyped(s.CmdDisp, cmdCompleteTask,
		func(ctx context.Context, cmd CompleteTaskCmd) error {
			return s.Repo.Execute(ctx, cmd.AggregateID(), aggregateType,
				Complete(CompleteTask{ID: cmd.AggregateID()}))
		}))

	must(command.RegisterTyped(s.CmdDisp, cmdArchiveTask,
		func(ctx context.Context, cmd ArchiveTaskCmd) error {
			return s.Repo.Execute(ctx, cmd.AggregateID(), aggregateType,
				Archive(ArchiveTask{ID: cmd.AggregateID()}))
		}))

	must(command.RegisterTyped(s.CmdDisp, cmdDeleteTask,
		func(ctx context.Context, cmd DeleteTaskCmd) error {
			return s.Repo.Execute(ctx, cmd.AggregateID(), aggregateType,
				Delete(DeleteTask{ID: cmd.AggregateID()}))
		}))

	must(command.RegisterTyped(s.CmdDisp, cmdUpdateTitle,
		func(ctx context.Context, cmd UpdateTitleCmd) error {
			return s.Repo.Execute(ctx, cmd.AggregateID(), aggregateType,
				UpdateTaskTitle(UpdateTitle{ID: cmd.AggregateID(), Title: cmd.Title}))
		}))

	must(command.RegisterTyped(s.CmdDisp, cmdChangePrio,
		func(ctx context.Context, cmd ChangePriorityCmd) error {
			return s.Repo.Execute(ctx, cmd.AggregateID(), aggregateType,
				ChangeTaskPriority(ChangePriority{ID: cmd.AggregateID(), Priority: cmd.Priority}))
		}))

	must(command.RegisterTyped(s.CmdDisp, cmdSetDueDate,
		func(ctx context.Context, cmd SetDueDateCmd) error {
			return s.Repo.Execute(ctx, cmd.AggregateID(), aggregateType,
				SetTaskDueDate(SetDueDate{ID: cmd.AggregateID(), DueDate: cmd.DueDate}))
		}))

	must(command.RegisterTyped(s.CmdDisp, cmdAddBlocker,
		func(ctx context.Context, cmd AddBlockerCmd) error {
			return s.Repo.Execute(ctx, cmd.AggregateID(), aggregateType,
				AddBlocker(BlockBy{ID: cmd.AggregateID(), DependencyID: cmd.DependencyID}))
		}))
}

// mustCmd creates a BasicCommand, panicking on error (programming bug).
func mustCmd(cmdType command.Type, aggID id.AggregateID) *command.BasicCommand {
	cmd, err := command.New(cmdType, aggID)
	if err != nil {
		panic(err)
	}

	return cmd
}
