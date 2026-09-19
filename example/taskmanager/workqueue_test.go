package main

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	qsqlite "github.com/larsartmann/go-cqrs-lite/queue/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// TestWorkQueue_AssignmentSurvivesRestart proves the T22 point in isolation:
// an enqueued assignment is DURABLE. Close the queue database (simulated
// crash), reopen it, and the job is still there — claimable under a fresh
// lease. A fire-and-forget goroutine has no such guarantee.
func TestWorkQueue_AssignmentSurvivesRestart(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "queue.db")

	ctx := context.Background()

	store, err := qsqlite.Open[AssignmentJob](path)
	if err != nil {
		t.Fatalf("open queue: %v", err)
	}

	job := AssignmentJob{TaskID: "01JRESTART", AssigneeID: defaultAssignee}

	for range 2 { // dedup key converges: two enqueues, one task
		if _, err := store.Enqueue(ctx, task.New[AssignmentJob]{
			Type:     assignmentQueueType,
			Payload:  job,
			DedupKey: "assign:" + job.TaskID,
		}); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close queue: %v", err)
	}

	reopened, err := qsqlite.Open[AssignmentJob](path)
	if err != nil {
		t.Fatalf("reopen queue: %v", err)
	}

	t.Cleanup(func() { _ = reopened.Close() })

	claim, err := reopened.ClaimDue(ctx, queueOwner, claimLease)
	if err != nil {
		t.Fatalf("claim after restart: %v", err)
	}

	if claim.Task.Payload != job {
		t.Fatalf("claimed payload %+v, want %+v", claim.Task.Payload, job)
	}

	if _, err := reopened.ClaimDue(ctx, "other-worker", claimLease); !errors.Is(err, queue.ErrNoTaskDue) {
		t.Fatalf("second claimer: error = %v, want ErrNoTaskDue (lease fencing)", err)
	}
}

// TestWorkQueue_AutoAssignEndToEnd runs the full loop: task.created →
// deriver enqueues on the durable queue → worker claims → command dispatch
// → task.assigned projected into the read model.
func TestWorkQueue_AutoAssignEndToEnd(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	ctx := context.Background()

	taskID := id.NewStreamID()

	if err := srv.CmdDisp.Dispatch(ctx, CreateTaskCmd{
		BasicCommand: Must(command.New(cmdCreateTask, taskID)),
		Title:        "Queue-assigned task",
	}); err != nil {
		t.Fatalf("create task: %v", err)
	}

	waitForView(t, srv, taskID, func(v *TaskView) bool {
		return v.AssigneeID == defaultAssignee
	})
}
