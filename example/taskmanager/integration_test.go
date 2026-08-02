package main

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gomust "github.com/larsartmann/go-must"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestIntegration_FullLifecycle exercises the entire CQRS pipeline:
// HTTP → Command Dispatcher → Decider Repository → Event Store → EventBus →
// CatchUpSubscriber → Materialize → Read Model → HTTP.
func TestIntegration_FullLifecycle(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	ctx := context.Background()

	// ── Create a task via command dispatcher ──────────────────────────
	taskID := id.NewStreamID()

	if err := srv.CmdDisp.Dispatch(ctx, CreateTaskCmd{
		BasicCommand: gomust.Must(command.New(cmdCreateTask, taskID)),
		Title:        "Integration Test",
		Priority:     PriorityHigh,
	}); err != nil {
		t.Fatalf("create task: %v", err)
	}

	waitForView(t, srv, taskID, func(v *TaskView) bool {
		return v.Title == "Integration Test" && v.Priority == PriorityHigh
	})

	// Wait for the deriver's auto-assign to complete before dispatching
	// the next command. The deriver dispatches task.assign asynchronously
	// on task.created — if we send task.start before the assign commits,
	// we get an optimistic-concurrency version conflict.
	waitForView(t, srv, taskID, func(v *TaskView) bool {
		return v.AssigneeID == defaultAssignee
	})

	// ── Start the task ────────────────────────────────────────────────
	if err := srv.CmdDisp.Dispatch(ctx, StartTaskCmd{
		BasicCommand: gomust.Must(command.New(cmdStartTask, taskID)),
	}); err != nil {
		t.Fatalf("start task: %v", err)
	}

	waitForView(t, srv, taskID, func(v *TaskView) bool {
		return v.Status == StatusActive
	})

	// ── Complete the task ─────────────────────────────────────────────
	if err := srv.CmdDisp.Dispatch(ctx, CompleteTaskCmd{
		BasicCommand: gomust.Must(command.New(cmdCompleteTask, taskID)),
	}); err != nil {
		t.Fatalf("complete task: %v", err)
	}

	waitForView(t, srv, taskID, func(v *TaskView) bool {
		return v.Status == StatusCompleted
	})

	// ── Verify via read model ─────────────────────────────────────────
	view, err := srv.Mat.View(ctx, taskID)
	if err != nil {
		t.Fatalf("view task: %v", err)
	}

	if view.Title != "Integration Test" {
		t.Errorf("title: got %q, want %q", view.Title, "Integration Test")
	}

	if view.Status != StatusCompleted {
		t.Errorf("status: got %q, want %q", view.Status, StatusCompleted)
	}

	// ── Delete the task (tombstone) ───────────────────────────────────
	// The tombstone event flows through the bus to the projection,
	// which sets Tombstoned=true on the view. We verify BOTH the
	// read-model projection AND the event-store metadata.
	if err := srv.CmdDisp.Dispatch(ctx, DeleteTaskCmd{
		BasicCommand: gomust.Must(command.New(cmdDeleteTask, taskID)),
	}); err != nil {
		t.Fatalf("delete task: %v", err)
	}

	// Verify the projection reflects the tombstone (not just the event store).
	waitForView(t, srv, taskID, func(v *TaskView) bool {
		return v.IsTombstoned()
	})

	// Verify the event store persisted correct tombstone metadata.
	ref := id.NewStreamRef(streamType, taskID)
	allEvents, err := srv.Bundle.EventSource.Load(ctx, ref)
	if err != nil {
		t.Fatalf("load events: %v", err)
	}

	last := allEvents[len(allEvents)-1]
	if last.Type() != evtTaskDeleted {
		t.Errorf("last event type: got %s, want %s", last.Type(), evtTaskDeleted)
	}

	if md := last.Metadata(); md.Tombstone == nil ||
		!md.Tombstone.Status.IsTombstoned() {
		t.Error("last event should have tombstone=tombstoned metadata")
	}
}

// TestIntegration_HTTPAPI verifies the HTTP endpoints work end-to-end.
func TestIntegration_HTTPAPI(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	ts := httptest.NewServer(srv.routes())
	defer ts.Close()

	// ── Create via HTTP ───────────────────────────────────────────────
	body := strings.NewReader(`{"title":"HTTP Task","priority":"urgent"}`)
	resp, err := http.Post(ts.URL+"/api/tasks", "application/json", body)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status: got %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var createResp struct {
		ID string `json:"id"`
	}
	_ = json.UnmarshalRead(resp.Body, &createResp)

	if createResp.ID == "" {
		t.Fatal("create: empty task ID")
	}

	taskID, _ := id.ParseStreamID(createResp.ID)

	// ── Wait for projection ───────────────────────────────────────────
	waitForView(t, srv, taskID, func(v *TaskView) bool {
		return v.Title == "HTTP Task"
	})

	// ── Get via HTTP ──────────────────────────────────────────────────
	resp2, err := http.Get(ts.URL + "/api/tasks/" + createResp.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("get status: got %d, want %d", resp2.StatusCode, http.StatusOK)
	}

	var task TaskView
	_ = json.UnmarshalRead(resp2.Body, &task)

	if task.Title != "HTTP Task" {
		t.Errorf("title: got %q, want %q", task.Title, "HTTP Task")
	}

	if task.Priority != PriorityUrgent {
		t.Errorf("priority: got %q, want %q", task.Priority, PriorityUrgent)
	}

	// ── List via HTTP ─────────────────────────────────────────────────
	resp3, err := http.Get(ts.URL + "/api/tasks")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer func() { _ = resp3.Body.Close() }()

	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("list status: got %d, want %d", resp3.StatusCode, http.StatusOK)
	}

	var listResp struct {
		Tasks []*TaskView `json:"tasks"`
		Count int         `json:"count"`
	}
	_ = json.UnmarshalRead(resp3.Body, &listResp)

	if listResp.Count < 1 {
		t.Errorf("list count: got %d, want >= 1", listResp.Count)
	}

	// ── Health check ──────────────────────────────────────────────────
	resp4, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	defer func() { _ = resp4.Body.Close() }()

	if resp4.StatusCode != http.StatusOK {
		t.Errorf("health status: got %d, want %d", resp4.StatusCode, http.StatusOK)
	}
}

// TestIntegration_MetaEngineTaskReader verifies the metaengine Map ADT
// projection serves real read-model queries: point lookup via Get and
// filtered scan via Scan with WithFilter on the "status" JSON field.
// This is the query that replaces the O(N) Materialize.List + Go-side filter.
func TestIntegration_MetaEngineTaskReader(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	ctx := context.Background()

	// Create two tasks.
	task1 := id.NewStreamID()
	if err := srv.CmdDisp.Dispatch(ctx, CreateTaskCmd{
		BasicCommand: gomust.Must(command.New(cmdCreateTask, task1)),
		Title:        "Metaengine Task 1", Priority: PriorityHigh,
	}); err != nil {
		t.Fatalf("create task1: %v", err)
	}

	task2 := id.NewStreamID()
	if err := srv.CmdDisp.Dispatch(ctx, CreateTaskCmd{
		BasicCommand: gomust.Must(command.New(cmdCreateTask, task2)),
		Title:        "Metaengine Task 2", Priority: PriorityMedium,
	}); err != nil {
		t.Fatalf("create task2: %v", err)
	}

	// Wait for both to appear in the metaengine projection.
	waitForMetaEngineView(
		t,
		srv,
		task1,
		func(v *TaskView) bool { return v.Title == "Metaengine Task 1" },
	)
	waitForMetaEngineView(
		t,
		srv,
		task2,
		func(v *TaskView) bool { return v.Title == "Metaengine Task 2" },
	)

	// ── Point lookup via TaskReader.Get ──
	view, found, err := srv.TaskReader.Get(ctx, task1.String())
	if err != nil {
		t.Fatalf("TaskReader.Get: %v", err)
	}

	if !found {
		t.Fatal("TaskReader.Get: task1 not found")
	}

	if view.Title != "Metaengine Task 1" {
		t.Errorf("title: got %q, want %q", view.Title, "Metaengine Task 1")
	}

	if view.Status != StatusPending {
		t.Errorf("status: got %q, want %q", view.Status, StatusPending)
	}

	// ── Scan all (no filter) ──
	all, err := srv.TaskReader.Scan(ctx)
	if err != nil {
		t.Fatalf("TaskReader.Scan all: %v", err)
	}

	if len(all) < 2 {
		t.Errorf("scan all: got %d, want >= 2", len(all))
	}

	// ── Start task1, then filter by status=active ──
	waitForView(t, srv, task1, func(v *TaskView) bool { return v.AssigneeID == defaultAssignee })

	if err := srv.CmdDisp.Dispatch(ctx, StartTaskCmd{
		BasicCommand: gomust.Must(command.New(cmdStartTask, task1)),
	}); err != nil {
		t.Fatalf("start task1: %v", err)
	}

	waitForMetaEngineView(t, srv, task1, func(v *TaskView) bool { return v.Status == StatusActive })

	// Filter by status=active — should find task1 but not task2 (still pending).
	active, err := srv.TaskReader.Scan(
		ctx,
		metaengine.WithFilter("status", metaengine.FilterEq, string(StatusActive)),
	)
	if err != nil {
		t.Fatalf("TaskReader.Scan active: %v", err)
	}

	foundTask1 := false

	for _, v := range active {
		if v.ID == task1.String() {
			foundTask1 = true
		}

		if v.Status != StatusActive {
			t.Errorf("active filter returned non-active task: %s status=%s", v.ID, v.Status)
		}
	}

	if !foundTask1 {
		t.Error("active filter: task1 not found in results")
	}

	// Filter by status=pending — should find task2 but not task1 (now active).
	pending, err := srv.TaskReader.Scan(
		ctx,
		metaengine.WithFilter("status", metaengine.FilterEq, string(StatusPending)),
	)
	if err != nil {
		t.Fatalf("TaskReader.Scan pending: %v", err)
	}

	foundTask2 := false

	for _, v := range pending {
		if v.ID == task2.String() {
			foundTask2 = true
		}

		if v.Status != StatusPending {
			t.Errorf("pending filter returned non-pending task: %s status=%s", v.ID, v.Status)
		}
	}

	if !foundTask2 {
		t.Error("pending filter: task2 not found in results")
	}
}

// waitForMetaEngineView polls the metaengine TaskReader until the task view
// matches the predicate or times out.
func waitForMetaEngineView(
	t *testing.T,
	srv *Server,
	taskID id.StreamID,
	matches func(*TaskView) bool,
) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		view, found, err := srv.TaskReader.Get(context.Background(), taskID.String())
		if err == nil && found && matches(&view) {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for metaengine view %s", taskID)
}

// ── Helpers ────────────────────────────────────────────────────────────────

func newTestServer(t *testing.T) *Server {
	t.Helper()

	srv, err := NewServer(DefaultConfig(), nil)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	t.Cleanup(func() { _ = srv.Stop() })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	srv.Start(ctx)

	// Give projection a moment to start
	time.Sleep(50 * time.Millisecond)

	return srv
}

func waitForView(t *testing.T, srv *Server, taskID id.StreamID, matches func(*TaskView) bool) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		view, err := srv.Mat.View(context.Background(), taskID)
		if err == nil && view != nil && matches(view) {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for view %s", taskID)
}
