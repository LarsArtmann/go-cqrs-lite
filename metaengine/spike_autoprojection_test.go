package metaengine

// Spike S1: Validate the fold inference API for v5 auto-projection.
//
// This throwaway test validates that system.View[V,K](name).From(samples...)
// can generate working folds. It discovers and solves the critical event-type
// string mismatch: AutoCRUDByConvention uses Go struct names ("TaskCreated")
// but the system pipeline uses dot-separated event types ("task.created").
//
// Findings are documented at the bottom of the file.

import (
	"context"
	"encoding/json/v2"
	"testing"
)

// ── Spike domain types ──

type spikeTaskCreated struct {
	ID     string
	Title  string
	Status string
}

type spikeTaskUpdated struct {
	ID     string
	Title  string
	Status string
}

type spikeTaskDeleted struct {
	ID string
}

type spikeTaskView struct {
	ID     string
	Title  string
	Status string
}

type spikeTaskQuery struct {
	ID string
}

// ── Test 1: AutoCRUDByConvention generates 3 folds with struct-name events ──

func TestSpike_AutoCRUDByConvention_GeneratesFolds(t *testing.T) {
	folds, err := AutoCRUDByConvention[spikeTaskView]("ID",
		spikeTaskCreated{}, spikeTaskUpdated{}, spikeTaskDeleted{},
	)
	if err != nil {
		t.Fatalf("AutoCRUDByConvention: %v", err)
	}

	if len(folds) != 3 {
		t.Fatalf("expected 3 folds, got %d", len(folds))
	}

	// Fold kinds: insert, update, remove
	kinds := []FoldKind{folds[0].Kind(), folds[1].Kind(), folds[2].Kind()}
	if kinds[0] != FoldInsert || kinds[1] != FoldUpdate || kinds[2] != FoldRemove {
		t.Fatalf("expected insert/update/remove, got %v", kinds)
	}

	// CRITICAL FINDING: fold EventType() = Go struct name, NOT a dot-separated type
	for _, f := range folds {
		t.Logf("fold kind=%s eventType=%q sample=%T", f.Kind(), f.EventType(), f.EventSample())
	}

	if folds[0].EventType() != "spikeTaskCreated" {
		t.Errorf("expected eventType 'spikeTaskCreated', got %q", folds[0].EventType())
	}
}

// ── Test 2: Direct Apply works with struct-name event types ──

func TestSpike_AutoCRUD_DirectApply(t *testing.T) {
	ctx := context.Background()

	folds, err := AutoCRUDByConvention[spikeTaskView]("ID",
		spikeTaskCreated{}, spikeTaskUpdated{}, spikeTaskDeleted{},
	)
	if err != nil {
		t.Fatal(err)
	}

	foldArgs := make([]any, len(folds))
	for i, f := range folds {
		foldArgs[i] = f
	}

	query := Query[spikeTaskQuery, spikeTaskView]("spike_tasks", foldArgs...)

	store, err := Plan([]Engine{NewMemoryEngine()}, query)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = store.Close() }()

	// Apply events using struct-name event types (matches fold EventType)
	if err := store.Apply(ctx, "spikeTaskCreated", spikeTaskCreated{
		ID: "t1", Title: "Spike task", Status: "open",
	}); err != nil {
		t.Fatalf("apply create: %v", err)
	}

	result, err := ExecuteTyped[spikeTaskQuery, spikeTaskView](ctx, store, spikeTaskQuery{ID: "t1"})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if result.Title != "Spike task" || result.Status != "open" {
		t.Fatalf("unexpected result: %+v", result)
	}

	// Update
	if err := store.Apply(ctx, "spikeTaskUpdated", spikeTaskUpdated{
		ID: "t1", Status: "done",
	}); err != nil {
		t.Fatalf("apply update: %v", err)
	}

	result, _ = ExecuteTyped[spikeTaskQuery, spikeTaskView](ctx, store, spikeTaskQuery{ID: "t1"})
	if result.Status != "done" {
		t.Fatalf("expected status 'done', got %q", result.Status)
	}

	t.Logf("✅ Direct apply with struct-name events works. Result: %+v", result)
}

// ── Test 3: The mismatch discovery ──
// AutoCRUDByConvention registers folds under "spikeTaskCreated".
// But the system pipeline creates events with event.New("task.created", ...).
// The fold lookup uses the event type string from the event, which won't match.

func TestSpike_EventTypeMismatch(t *testing.T) {
	folds, err := AutoCRUDByConvention[spikeTaskView]("ID",
		spikeTaskCreated{}, spikeTaskUpdated{}, spikeTaskDeleted{},
	)
	if err != nil {
		t.Fatal(err)
	}

	// The folds are registered under struct names
	registeredTypes := make(map[string]bool)
	for _, f := range folds {
		registeredTypes[f.EventType()] = true
	}

	// The system pipeline uses dot-separated event types
	wireTypes := []string{"task.created", "task.updated", "task.deleted"}

	mismatch := false
	for _, wt := range wireTypes {
		if !registeredTypes[wt] {
			t.Logf("❌ MISMATCH: wire event type %q has no matching fold", wt)
			mismatch = true
		}
	}

	if !mismatch {
		t.Fatal("expected event type mismatch but found none")
	}

	t.Log("✅ Confirmed: AutoCRUDByConvention uses struct names, system pipeline uses dot-separated types")
	t.Log("   Solution: need a variant that accepts explicit event type strings")
}

// ── Test 5: Validate AutoCRUDByNamedEvents (the real exported function) ──

func TestSpike_AutoCRUDByNamedEvents_Works(t *testing.T) {
	ctx := context.Background()

	folds, err := AutoCRUDByNamedEvents[spikeTaskView]("ID",
		NamedEvent("task.created", spikeTaskCreated{}),
		NamedEvent("task.updated", spikeTaskUpdated{}),
		NamedEvent("task.deleted", spikeTaskDeleted{}),
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(folds) != 3 {
		t.Fatalf("expected 3 folds, got %d", len(folds))
	}

	// Verify event types are now dot-separated
	if folds[0].EventType() != "task.created" {
		t.Errorf("expected 'task.created', got %q", folds[0].EventType())
	}

	if folds[1].EventType() != "task.updated" {
		t.Errorf("expected 'task.updated', got %q", folds[1].EventType())
	}

	if folds[2].EventType() != "task.deleted" {
		t.Errorf("expected 'task.deleted', got %q", folds[2].EventType())
	}

	// Build query and store
	foldArgs := make([]any, len(folds))
	for i, f := range folds {
		foldArgs[i] = f
	}

	query := Query[spikeTaskQuery, spikeTaskView]("spike_tasks_named", foldArgs...)

	store, err := Plan([]Engine{NewMemoryEngine()}, query)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = store.Close() }()

	// Apply with dot-separated event types — this is what the system pipeline does
	if err := store.Apply(ctx, "task.created", spikeTaskCreated{
		ID: "t2", Title: "Named event task", Status: "open",
	}); err != nil {
		t.Fatalf("apply create: %v", err)
	}

	result, err := ExecuteTyped[spikeTaskQuery, spikeTaskView](ctx, store, spikeTaskQuery{ID: "t2"})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if result.Title != "Named event task" {
		t.Fatalf("unexpected result: %+v", result)
	}

	// Update
	if err := store.Apply(ctx, "task.updated", spikeTaskUpdated{
		ID: "t2", Status: "in_progress",
	}); err != nil {
		t.Fatalf("apply update: %v", err)
	}

	result, _ = ExecuteTyped[spikeTaskQuery, spikeTaskView](ctx, store, spikeTaskQuery{ID: "t2"})
	if result.Status != "in_progress" {
		t.Fatalf("expected 'in_progress', got %q", result.Status)
	}

	// Delete
	if err := store.Apply(ctx, "task.deleted", spikeTaskDeleted{ID: "t2"}); err != nil {
		t.Fatalf("apply delete: %v", err)
	}

	_, err = ExecuteTyped[spikeTaskQuery, spikeTaskView](ctx, store, spikeTaskQuery{ID: "t2"})
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}

	t.Logf("✅ AutoCRUDByNamedEvents works with dot-separated event types: %+v", result)
}

// ── Test 6: Full JSON round-trip (simulates projectionadapter flow) ──
// The adapter JSON-encodes the payload, then JSON-decodes it before applying.
// The decoded payload type must match what the fold expects.

func TestSpike_AutoCRUDByNamedEvents_JSONRoundTrip(t *testing.T) {
	ctx := context.Background()

	folds, err := AutoCRUDByNamedEvents[spikeTaskView]("ID",
		NamedEvent("task.created", spikeTaskCreated{}),
		NamedEvent("task.updated", spikeTaskUpdated{}),
		NamedEvent("task.deleted", spikeTaskDeleted{}),
	)
	if err != nil {
		t.Fatal(err)
	}

	foldArgs := make([]any, len(folds))
	for i, f := range folds {
		foldArgs[i] = f
	}

	query := Query[spikeTaskQuery, spikeTaskView]("spike_json", foldArgs...)

	store, err := Plan([]Engine{NewMemoryEngine()}, query)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = store.Close() }()

	// Simulate: encode payload → decode → apply (what projectionadapter does)
	original := spikeTaskCreated{ID: "t3", Title: "JSON round-trip", Status: "open"}

	payloadBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded spikeTaskCreated
	if err := json.Unmarshal(payloadBytes, &decoded); err != nil {
		t.Fatal(err)
	}

	// Apply with the decoded payload
	if err := store.Apply(ctx, "task.created", decoded); err != nil {
		t.Fatalf("apply: %v", err)
	}

	result, err := ExecuteTyped[spikeTaskQuery, spikeTaskView](ctx, store, spikeTaskQuery{ID: "t3"})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if result.Title != "JSON round-trip" {
		t.Fatalf("unexpected: %+v", result)
	}

	t.Logf("✅ JSON round-trip works. Result: %+v", result)
}

// ── Test 7: Fold quality — field matching behavior ──

func TestSpike_FieldMatching(t *testing.T) {
	type partialView struct {
		ID    string
		Title string // no Status field
	}

	folds, err := AutoCRUDByNamedEvents[partialView]("ID",
		NamedEvent("task.created", spikeTaskCreated{}),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	foldArgs := make([]any, len(folds))
	for i, f := range folds {
		foldArgs[i] = f
	}

	query := Query[struct{ ID string }, partialView]("spike_partial", foldArgs...)

	store, err := Plan([]Engine{NewMemoryEngine()}, query)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = store.Close() }()

	if err := store.Apply(ctx, "task.created", spikeTaskCreated{
		ID: "p1", Title: "Partial", Status: "ignored",
	}); err != nil {
		t.Fatal(err)
	}

	result, err := ExecuteTyped[struct{ ID string }, partialView](ctx, store, struct{ ID string }{ID: "p1"})
	if err != nil {
		t.Fatal(err)
	}

	if result.Title != "Partial" {
		t.Fatalf("expected 'Partial', got %q", result.Title)
	}

	t.Logf("✅ Field matching: only shared fields copied. Status is ignored. Result: %+v", result)
}

// ── SPIKE FINDINGS ──
//
// 1. AutoCRUDByConvention WORKS for fold generation. Field matching (matchFields)
//    correctly maps shared exported fields between event and view types.
//
// 2. CRITICAL PROBLEM: AutoCRUDByConvention uses Go struct names as event types
//    (EventTypeName = reflect.TypeOf(sample).Name()). But the system pipeline
//    uses dot-separated event types ("task.created"). Folds registered under
//    "spikeTaskCreated" never match events with type "task.created".
//
// 3. SOLUTION (validated): Add AutoCRUDByNamedEvents[R](keyField, ...EventSample)
//    that pairs wire event type strings with sample structs. Implementation:
//    call existing autoInsertByType/autoUpdateByType/autoDeleteByType, then
//    override the eventType field on the resulting fold struct. ~30 lines of
//    new code in metaengine.
//
// 4. The override works because autoInsertByType etc. return concrete *insertFold
//    etc. pointers, and eventType is an unexported field accessible within the
//    metaengine package. No structural change to Fold interface needed.
//
// 5. JSON round-trip works: the projectionadapter JSON-encodes event payloads,
//    JSON-decodes them, and passes the decoded struct to store.Apply. The fold's
//    invoke closure uses reflection to extract fields, which works on any value
//    of the correct type regardless of how it was constructed.
//
// 6. Field matching is conservative: only fields present in BOTH event and view
//    with matching names and assignable types are copied. Extra fields in either
//    direction are silently ignored. This is the right default for auto-projection.
//
// 7. The system/projection_builder.go (T04) should:
//    a. Accept ProjectionSpec{Name, ResultType, KeyField, Events []EventSample}
//    b. Call AutoCRUDByNamedEvents (the new function)
//    c. Wrap folds in Query[InputType, ResultType](name, foldArgs...)
//    d. Build TypeDecoder entries: for each EventSample, RegisterString(eventType, sample)
//    e. Return (QueryDecl-as-any, *TypeDecoder) for system.New() to consume
//
// API for the v5 consumer:
//   system.View[TaskView, TaskID]("tasks").
//       From(system.Event("task.created", TaskCreated{}),
//            system.Event("task.updated", TaskUpdated{}),
//            system.Event("task.deleted", TaskDeleted{}))
//
// Or the simpler form for convention-matching event types:
//   system.View[TaskView, TaskID]("tasks").
//       From("task.created", TaskCreated{},
//            "task.updated", TaskUpdated{},
//            "task.deleted", TaskDeleted{})
