package metaengine_test

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ─── DOMAIN TYPES (shared across BDD specs) ───
// A task management domain exercising all 5 ADTs.

type TaskID string
type UserID string

// Event types.
type TaskCreated struct {
	ID       TaskID
	Title    string
	Assignee UserID
	Status   string
	Priority int
	At       time.Time
}

type TaskCompleted struct {
	ID TaskID
	At time.Time
}

type TaskDeleted struct {
	ID TaskID
	At time.Time
}

type TaskAssigned struct {
	TaskID    TaskID
	Assignee  UserID
	Previous  UserID
	At        time.Time
}

// ─── Query 1: FindTask (Map ADT — point lookup) ───

type FindTask struct {
	ID TaskID
}

type FindTaskResult struct {
	ID       TaskID
	Title    string
	Assignee UserID
	Status   string
	Priority int
}

func findTaskQuery() metaengine.QueryDecl[FindTask, FindTaskResult] {
	return metaengine.Query[FindTask, FindTaskResult](
		"find_task",
		metaengine.On(TaskCreated{}, func(e TaskCreated) (TaskID, FindTaskResult) {
			return e.ID, FindTaskResult{
				ID: e.ID, Title: e.Title, Assignee: e.Assignee,
				Status: e.Status, Priority: e.Priority,
			}
		}),
		metaengine.On(TaskCompleted{}, func(e TaskCompleted, prev FindTaskResult) FindTaskResult {
			prev.Status = "completed"
			return prev
		}),
		metaengine.On(TaskDeleted{}, metaengine.Remove[FindTaskResult]()),
	)
}

// ─── Query 2: CheckAssignee (Set ADT — membership) ───

type CheckAssignee struct {
	User UserID
}

func checkAssigneeQuery() metaengine.QueryDecl[CheckAssignee, bool] {
	return metaengine.Query[CheckAssignee, bool](
		"check_assignee",
		metaengine.On(TaskAssigned{}, func(e TaskAssigned) UserID {
			return e.Assignee
		}),
	)
}

// ─── Query 3: ListTasksByStatus (SortedMap ADT — filtered scan) ───

type ListTasksByStatus struct {
	Status string
	Limit  int
	After  *metaengine.Cursor
}

type ListTasksByStatusResult struct {
	Tasks []FindTaskResult
	Next  *metaengine.Cursor
}

func listTasksByStatusQuery() metaengine.QueryDecl[ListTasksByStatus, ListTasksByStatusResult] {
	return metaengine.Query[ListTasksByStatus, ListTasksByStatusResult](
		"list_tasks_by_status",
		metaengine.On(TaskCreated{}, func(e TaskCreated) (TaskID, FindTaskResult) {
			return e.ID, FindTaskResult{
				ID: e.ID, Title: e.Title, Assignee: e.Assignee,
				Status: e.Status, Priority: e.Priority,
			}
		}),
		metaengine.On(TaskCompleted{}, func(e TaskCompleted, prev FindTaskResult) FindTaskResult {
			prev.Status = "completed"
			return prev
		}),
		metaengine.On(TaskDeleted{}, metaengine.Remove[FindTaskResult]()),
		metaengine.FilterOn(func(r FindTaskResult) string { return r.Status }),
		metaengine.SortOn(func(r FindTaskResult) int { return r.Priority }),
	)
}

// ─── Query 4: CountByStatus (Counter ADT — aggregate) ───

type CountByStatus struct{}

func countByStatusQuery() metaengine.QueryDecl[CountByStatus, map[string]int64] {
	return metaengine.Query[CountByStatus, map[string]int64](
		"count_by_status",
		metaengine.On(TaskCreated{}, func(e TaskCreated) metaengine.Delta {
			return metaengine.Delta{e.Status: +1}
		}),
		metaengine.On(TaskCompleted{}, func(e TaskCompleted) metaengine.Delta {
			return metaengine.Delta{"open": -1, "completed": +1}
		}),
	)
}

// ─── Query 5: TasksByAssignee (Graph ADT — traversal) ───

type TasksByAssignee struct {
	User  UserID
	Depth int
}

type TasksByAssigneeResult struct {
	TaskIDs []TaskID
}

func tasksByAssigneeQuery() metaengine.QueryDecl[TasksByAssignee, TasksByAssigneeResult] {
	return metaengine.Query[TasksByAssignee, TasksByAssigneeResult](
		"tasks_by_assignee",
		metaengine.On(TaskAssigned{}, func(e TaskAssigned) metaengine.Edge {
			return metaengine.Edge{From: e.Assignee, To: e.TaskID}
		}),
	)
}

// allQueries returns all 5 query declarations for multi-query tests.
func allQueries() []any {
	return []any{
		findTaskQuery(),
		checkAssigneeQuery(),
		listTasksByStatusQuery(),
		countByStatusQuery(),
		tasksByAssigneeQuery(),
	}
}
