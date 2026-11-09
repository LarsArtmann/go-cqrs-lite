package metaengine_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// TestAutoCRUDByConvention_BasicSuccess verifies suffix-based type inference
// for Created/Updated/Deleted event types.
func TestAutoCRUDByConvention_BasicSuccess(t *testing.T) {
	t.Parallel()

	type convCreated struct {
		ID   string
		Name string
	}

	type convUpdated struct {
		ID   string
		Name string
	}

	type convDeleted struct {
		ID string
	}

	type convView struct {
		ID   string
		Name string
	}

	type convQuery struct {
		ID string
	}

	folds, err := metaengine.AutoCRUDByConvention[convView]("ID",
		convCreated{}, convUpdated{}, convDeleted{},
	)
	if err != nil {
		t.Fatalf("AutoCRUDByConvention: %v", err)
	}

	if len(folds) != 3 {
		t.Fatalf("expected 3 folds, got %d", len(folds))
	}

	foldArgs := make([]any, len(folds))
	for i, f := range folds {
		foldArgs[i] = f
	}

	q := metaengine.Query[convQuery, convView]("conv", foldArgs...)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Insert
	rec := record.Record{
		Type:       "convCreated",
		StreamID:   record.NewStreamRef("Conv", "c1"),
		StreamType: "Conv",
		Version:    1,
	}
	if err := store.ApplyRecord(ctx, rec, convCreated{ID: "x1", Name: "Alpha"}); err != nil {
		t.Fatalf("ApplyRecord insert: %v", err)
	}

	result, err := metaengine.ExecuteTyped[convQuery, convView](ctx, store, convQuery{ID: "x1"})
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if result.Name != "Alpha" {
		t.Errorf("Name = %q, want Alpha", result.Name)
	}

	// Update
	rec2 := record.Record{
		Type:       "convUpdated",
		StreamID:   record.NewStreamRef("Conv", "c1"),
		StreamType: "Conv",
		Version:    2,
	}
	if err := store.ApplyRecord(ctx, rec2, convUpdated{ID: "x1", Name: "Beta"}); err != nil {
		t.Fatalf("ApplyRecord update: %v", err)
	}

	result, err = metaengine.ExecuteTyped[convQuery, convView](ctx, store, convQuery{ID: "x1"})
	if err != nil {
		t.Fatalf("ExecuteTyped after update: %v", err)
	}

	if result.Name != "Beta" {
		t.Errorf("Name after update = %q, want Beta", result.Name)
	}

	// Delete
	rec3 := record.Record{
		Type:       "convDeleted",
		StreamID:   record.NewStreamRef("Conv", "c1"),
		StreamType: "Conv",
		Version:    3,
	}
	if err := store.ApplyRecord(ctx, rec3, convDeleted{ID: "x1"}); err != nil {
		t.Fatalf("ApplyRecord delete: %v", err)
	}

	_, err = metaengine.ExecuteTyped[convQuery, convView](ctx, store, convQuery{ID: "x1"})
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

// TestAutoCRUDByConvention_MissingCreated verifies that omitting the Created
// type returns an error (insert is the minimum requirement).
func TestAutoCRUDByConvention_MissingCreated(t *testing.T) {
	t.Parallel()

	type convUpdated struct {
		ID string
	}

	type convDeleted struct {
		ID string
	}

	type convView struct {
		ID string
	}

	_, err := metaengine.AutoCRUDByConvention[convView]("ID",
		convUpdated{}, convDeleted{},
	)
	if err == nil {
		t.Fatal("expected error when no Created sample is provided")
	}
}

// TestAutoCRUDByConvention_UnrecognizedSuffix verifies that event types
// without a recognized suffix return an error.
func TestAutoCRUDByConvention_UnrecognizedSuffix(t *testing.T) {
	t.Parallel()

	type weirdEvent struct {
		ID string
	}

	type convView struct {
		ID string
	}

	_, err := metaengine.AutoCRUDByConvention[convView]("ID",
		weirdEvent{},
	)
	if err == nil {
		t.Fatal("expected error for unrecognized suffix")
	}
}

// TestAutoCRUDByConvention_DuplicateSuffix verifies that providing two samples
// with the same suffix returns an error.
func TestAutoCRUDByConvention_DuplicateSuffix(t *testing.T) {
	t.Parallel()

	type firstCreated struct {
		ID string
	}

	type secondCreated struct {
		ID string
	}

	type convView struct {
		ID string
	}

	_, err := metaengine.AutoCRUDByConvention[convView]("ID",
		firstCreated{}, secondCreated{},
	)
	if err == nil {
		t.Fatal("expected error for duplicate Created types")
	}
}

// TestAutoCRUDByConvention_OnlyCreated verifies that providing only a Created
// type works (generates just the insert fold).
func TestAutoCRUDByConvention_OnlyCreated(t *testing.T) {
	t.Parallel()

	type onlyCreated struct {
		ID   string
		Name string
	}

	type onlyView struct {
		ID   string
		Name string
	}

	folds, err := metaengine.AutoCRUDByConvention[onlyView]("ID",
		onlyCreated{},
	)
	if err != nil {
		t.Fatalf("AutoCRUDByConvention: %v", err)
	}

	if len(folds) != 1 {
		t.Fatalf("expected 1 fold (insert only), got %d", len(folds))
	}

	if folds[0].Kind() != metaengine.FoldInsert {
		t.Errorf("fold kind = %s, want insert", folds[0].Kind())
	}
}
