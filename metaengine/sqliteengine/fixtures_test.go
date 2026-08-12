package sqliteengine_test

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

type (
	TaskID string
	UserID string
)

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
	)
}
