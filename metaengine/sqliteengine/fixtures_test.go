package sqliteengine_test

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
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
		metaengine.OnRecord(TaskCreated{}, func(_ record.Record, e TaskCreated) (TaskID, FindTaskResult) {
			return e.ID, FindTaskResult{
				ID: e.ID, Title: e.Title, Assignee: e.Assignee,
				Status: e.Status, Priority: e.Priority,
			}
		}),
		metaengine.OnRecord(TaskCompleted{}, func(_ record.Record, e TaskCompleted, prev FindTaskResult) FindTaskResult {
			prev.Status = "completed"

			return prev
		}),
	)
}
