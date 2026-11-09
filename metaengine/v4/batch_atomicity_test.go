package metaengine_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func TestBatchAtomicity_MultipleQueriesSameEvent(t *testing.T) {
	t.Parallel()

	type UserID string
	type TaskID string

	type TaskCreated struct {
		ID     TaskID
		UserID UserID
		Title  string
	}

	type TaskByID struct {
		ID     TaskID
		UserID UserID
		Title  string
	}

	type TaskByUser struct {
		UserID UserID
		TaskID TaskID
		Title  string
	}

	type GetTask struct{ ID TaskID }
	type ListUserTasks struct{ UserID UserID }

	q1 := metaengine.Query[GetTask, TaskByID]("tasks_by_id",
		metaengine.OnRecord(TaskCreated{},
			func(_ record.Record, e TaskCreated) (TaskID, TaskByID) {
				return e.ID, TaskByID(e)
			}))

	q2 := metaengine.Query[ListUserTasks, TaskByUser]("tasks_by_user",
		metaengine.OnRecord(TaskCreated{},
			func(_ record.Record, e TaskCreated) (UserID, TaskByUser) {
				return e.UserID, TaskByUser{UserID: e.UserID, TaskID: e.ID, Title: e.Title}
			}))

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q1, q2)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	err = store.Apply(ctx, "TaskCreated", TaskCreated{ID: "t1", UserID: "u1", Title: "Test"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	task, err := metaengine.ExecuteTyped[GetTask, TaskByID](ctx, store, GetTask{ID: "t1"})
	if err != nil {
		t.Fatalf("ExecuteTyped tasks_by_id: %v", err)
	}

	if task.Title != "Test" {
		t.Errorf("tasks_by_id Title = %q, want %q", task.Title, "Test")
	}

	userTasks, err := metaengine.ExecuteTyped[ListUserTasks, TaskByUser](
		ctx, store, ListUserTasks{UserID: "u1"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped tasks_by_user: %v", err)
	}

	if userTasks.TaskID != "t1" {
		t.Errorf("tasks_by_user TaskID = %q, want %q", userTasks.TaskID, "t1")
	}
}

func TestBatchAtomicity_AllQueriesUpdatedBySingleEvent(t *testing.T) {
	t.Parallel()

	type ID string
	type Item struct {
		ID    ID
		Name  string
		Value int
	}

	type Created struct {
		ID    ID
		Name  string
		Value int
	}

	type GetByID struct{ ID ID }
	type GetByName struct{ Name string }

	// Two queries listening to the same event, on the same engine
	q1 := metaengine.Query[GetByID, Item]("items_by_id",
		metaengine.OnRecord(Created{},
			func(_ record.Record, e Created) (ID, Item) {
				return e.ID, Item(e)
			}))

	q2 := metaengine.Query[GetByName, Item]("items_by_name",
		metaengine.OnRecord(Created{},
			func(_ record.Record, e Created) (string, Item) {
				return e.Name, Item(e)
			}))

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q1, q2)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Apply one event — both collections should be updated atomically
	err = store.Apply(ctx, "Created", Created{ID: "x1", Name: "widget", Value: 42})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// Verify both collections have the data
	byID, err := metaengine.ExecuteTyped[GetByID, Item](ctx, store, GetByID{ID: "x1"})
	if err != nil {
		t.Fatalf("by_id lookup: %v", err)
	}

	if byID.Value != 42 {
		t.Errorf("by_id Value = %d, want 42", byID.Value)
	}

	byName, err := metaengine.ExecuteTyped[GetByName, Item](ctx, store, GetByName{Name: "widget"})
	if err != nil {
		t.Fatalf("by_name lookup: %v", err)
	}

	if byName.ID != "x1" {
		t.Errorf("by_name ID = %q, want %q", byName.ID, "x1")
	}
}
