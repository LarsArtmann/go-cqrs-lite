package projectionadapter_test

import (
	"context"
	"encoding/json/v2"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestIntegration_AutoInsert_ThroughAdapter is the end-to-end integration test
// proving the full Record-aware pipeline: an event flows through
// projectionadapter.Handle() → AutoInsert fold → Record metadata is stamped into
// the result → ExecuteTyped returns a view with StreamID, Version, and
// CorrelationID populated from the event (ADR-0112, ADR-0116).
func TestIntegration_AutoInsert_ThroughAdapter(t *testing.T) {
	t.Parallel()

	type taskCreated struct {
		ID     string
		Title  string
		Status string
	}

	type taskView struct {
		ID            string
		Title         string
		Status        string
		StreamID      string // auto-stamped from Record
		Version       int64  // auto-stamped from Record
		CorrelationID string // auto-stamped from Record
	}

	type taskQuery struct {
		ID string
	}

	q := metaengine.Query[taskQuery, taskView](
		"tasks",
		metaengine.AutoInsert[taskCreated, taskView]("ID"),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
	)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}
	defer store.Close()

	// Use a plain PayloadDecoder — AutoInsert expects the raw event type.
	decoder := func(eventType string, payload []byte) (any, error) {
		var e taskCreated
		err := json.Unmarshal(payload, &e)
		return e, err
	}

	adapter := projectionadapter.New("tasks-proj", store, decoder)

	// Create a real event with full metadata.
	streamID := id.NewStreamID()
	correlationID := id.NewCorrelationID()
	userID := id.NewUserID()

	payloadJSON, _ := json.Marshal(taskCreated{
		ID: "task-1", Title: "Build pipeline", Status: "open",
	})

	evt, err := event.NewEvent(
		"taskCreated", streamID, "Task", event.Version(3),
		payloadJSON,
		event.WithCorrelationID(correlationID),
		event.WithUserID(userID),
	)
	if err != nil {
		t.Fatalf("event.NewEvent: %v", err)
	}

	// Run through the adapter — this calls ApplyRecord internally.
	if err := adapter.Handle(context.Background(), evt); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	// Query the result — Record metadata should be stamped.
	result, err := metaengine.ExecuteTyped[taskQuery, taskView](
		context.Background(), store, taskQuery{ID: "task-1"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	// Verify event-mapped fields.
	if result.Title != "Build pipeline" {
		t.Errorf("Title = %q, want 'Build pipeline'", result.Title)
	}

	if result.Status != "open" {
		t.Errorf("Status = %q, want 'open'", result.Status)
	}

	// Verify auto-stamped Record metadata fields.
	wantStreamID := "Task/" + streamID.String()
	if result.StreamID != wantStreamID {
		t.Errorf("StreamID = %q, want %q", result.StreamID, wantStreamID)
	}

	if result.Version != 3 {
		t.Errorf("Version = %d, want 3", result.Version)
	}

	if result.CorrelationID != correlationID.String() {
		t.Errorf("CorrelationID = %q, want %q", result.CorrelationID, correlationID.String())
	}
}

// TestIntegration_AutoCRUD_FullLifecycle_ThroughAdapter proves the full CRUD
// lifecycle works through the adapter: create → update → query → verify update.
func TestIntegration_AutoCRUD_FullLifecycle_ThroughAdapter(t *testing.T) {
	t.Parallel()

	type docCreated struct {
		ID    string
		Name  string
		Value int64
	}

	type docUpdated struct {
		ID    string
		Name  string
		Value int64
	}

	type docDeleted struct {
		ID string
	}

	type docView struct {
		ID       string
		Name     string
		Value    int64
		StreamID string
		Version  int64
	}

	type docQuery struct {
		ID string
	}

	folds := metaengine.AutoCRUD[docCreated, docUpdated, docDeleted, docView]("ID")

	foldArgs := make([]any, len(folds))
	for i, f := range folds {
		foldArgs[i] = f
	}

	q := metaengine.Query[docQuery, docView]("docs", foldArgs...)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
	)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}
	defer store.Close()

	decoder := func(eventType string, payload []byte) (any, error) {
		switch eventType {
		case "docCreated":
			var e docCreated
			return e, json.Unmarshal(payload, &e)
		case "docUpdated":
			var e docUpdated
			return e, json.Unmarshal(payload, &e)
		case "docDeleted":
			var e docDeleted
			return e, json.Unmarshal(payload, &e)
		default:
			return nil, nil
		}
	}

	adapter := projectionadapter.New("docs-proj", store, decoder)

	ctx := context.Background()

	// 1. Create
	streamID := id.NewStreamID()
	createJSON, _ := json.Marshal(docCreated{ID: "d1", Name: "Original", Value: 10})
	createEvt, _ := event.NewEvent("docCreated", streamID, "Doc", event.Version(1), createJSON)
	if err := adapter.Handle(ctx, createEvt); err != nil {
		t.Fatalf("Handle create: %v", err)
	}

	result, err := metaengine.ExecuteTyped[docQuery, docView](ctx, store, docQuery{ID: "d1"})
	if err != nil {
		t.Fatalf("ExecuteTyped after create: %v", err)
	}

	if result.Name != "Original" || result.Value != 10 {
		t.Fatalf("after create: Name=%q Value=%d, want Original/10", result.Name, result.Value)
	}

	if result.Version != 1 {
		t.Errorf("Version after create = %d, want 1", result.Version)
	}

	// 2. Update
	updateJSON, _ := json.Marshal(docUpdated{ID: "d1", Name: "Updated", Value: 20})
	updateEvt, _ := event.NewEvent("docUpdated", streamID, "Doc", event.Version(2), updateJSON)
	if err := adapter.Handle(ctx, updateEvt); err != nil {
		t.Fatalf("Handle update: %v", err)
	}

	result, err = metaengine.ExecuteTyped[docQuery, docView](ctx, store, docQuery{ID: "d1"})
	if err != nil {
		t.Fatalf("ExecuteTyped after update: %v", err)
	}

	if result.Name != "Updated" || result.Value != 20 {
		t.Fatalf("after update: Name=%q Value=%d, want Updated/20", result.Name, result.Value)
	}

	if result.Version != 2 {
		t.Errorf("Version after update = %d, want 2", result.Version)
	}

	// 3. Delete
	deleteJSON, _ := json.Marshal(docDeleted{ID: "d1"})
	deleteEvt, _ := event.NewEvent("docDeleted", streamID, "Doc", event.Version(3), deleteJSON)
	if err := adapter.Handle(ctx, deleteEvt); err != nil {
		t.Fatalf("Handle delete: %v", err)
	}

	_, err = metaengine.ExecuteTyped[docQuery, docView](ctx, store, docQuery{ID: "d1"})
	if err == nil {
		t.Fatal("expected error after delete (not found), got nil")
	}
}
