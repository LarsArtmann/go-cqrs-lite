package main

import (
	"context"
	"testing"
	"time"

	gomust "github.com/larsartmann/go-must"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestMetaEngine_TaskCountsByStatus verifies the metaengine Counter projection
// tracks task lifecycle status transitions correctly. This is the showcase
// integration: an O(1) aggregate counter replacing the O(N) Materialize.List
// + Go-side status filter in handleListTasks.
func TestMetaEngine_TaskCountsByStatus(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	ctx := context.Background()

	taskA := id.NewStreamID()
	taskB := id.NewStreamID()
	taskC := id.NewStreamID()

	dispatch(t, srv, ctx, cmdCreateTask, taskA, "Task A", PriorityLow)
	dispatch(t, srv, ctx, cmdCreateTask, taskB, "Task B", PriorityMedium)
	dispatch(t, srv, ctx, cmdCreateTask, taskC, "Task C", PriorityHigh)

	waitForCounter(t, srv, "pending", 3)

	dispatch(t, srv, ctx, cmdStartTask, taskA, "", "")
	waitForCounter(t, srv, "pending", 2)
	waitForCounter(t, srv, "active", 1)

	dispatch(t, srv, ctx, cmdStartTask, taskB, "", "")
	waitForCounter(t, srv, "pending", 1)
	waitForCounter(t, srv, "active", 2)

	dispatch(t, srv, ctx, cmdCompleteTask, taskA, "", "")
	waitForCounter(t, srv, "active", 1)
	waitForCounter(t, srv, "completed", 1)

	dispatch(t, srv, ctx, cmdArchiveTask, taskA, "", "")
	waitForCounter(t, srv, "completed", 0)
	waitForCounter(t, srv, "archived", 1)
}

// ─── helpers ───

func dispatch(
	t *testing.T,
	srv *Server,
	ctx context.Context,
	cmdType command.Type,
	taskID id.StreamID,
	title string,
	priority Priority,
) {
	t.Helper()

	switch cmdType {
	case cmdCreateTask:
		err := srv.CmdDisp.Dispatch(ctx, CreateTaskCmd{
			BasicCommand: gomust.Must(command.New(cmdCreateTask, taskID)),
			Title:        title,
			Priority:     priority,
		})
		mustNoErr(t, err, "create task")

	case cmdStartTask:
		err := srv.CmdDisp.Dispatch(ctx, StartTaskCmd{
			BasicCommand: gomust.Must(command.New(cmdStartTask, taskID)),
		})
		mustNoErr(t, err, "start task")

	case cmdCompleteTask:
		err := srv.CmdDisp.Dispatch(ctx, CompleteTaskCmd{
			BasicCommand: gomust.Must(command.New(cmdCompleteTask, taskID)),
		})
		mustNoErr(t, err, "complete task")

	case cmdArchiveTask:
		err := srv.CmdDisp.Dispatch(ctx, ArchiveTaskCmd{
			BasicCommand: gomust.Must(command.New(cmdArchiveTask, taskID)),
		})
		mustNoErr(t, err, "archive task")
	}
}

func waitForCounter(t *testing.T, srv *Server, status string, want int64) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		counts, err := metaengine.ExecuteTyped[taskCountsInput, map[string]int64](
			context.Background(), srv.MetaEngine, taskCountsInput{},
		)
		if err == nil && counts[status] == want {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for counter %q = %d", status, want)
}

func mustNoErr(t *testing.T, err error, what string) {
	t.Helper()

	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
}
