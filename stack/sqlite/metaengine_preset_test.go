package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// TestPreset_WithMetaEngine verifies that a stack/sqlite bundle correctly
// wires a metaengine Store via stack.WithMetaEngine, exposes it via
// bundle.MetaEngine(), and supports Apply + ExecuteTyped end-to-end.
func TestPreset_WithMetaEngine(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dsn := filepath.Join(dir, "test.db")

	// Build the metaengine store before the bundle.
	type countInput struct{}

	countQ := metaengine.Query[countInput, map[string]int64](
		"preset_counts",
		metaengine.On(presetItemCreated{}, func(e presetItemCreated) metaengine.Delta {
			return metaengine.Delta{e.Status: +1}
		}),
	)

	store, err := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, countQ)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}

	bundle, err := sqlite.New(
		dsn,
		sqlite.WithStack(stack.WithMetaEngine(store)),
	)
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	// The bundle must expose the store via MetaEngine().
	if bundle.MetaEngine() == nil {
		t.Fatal("bundle.MetaEngine() is nil — WithMetaEngine did not wire")
	}

	// Apply events and execute the query through the preset path.
	ctx := context.Background()

	if err := bundle.MetaEngine().Apply(ctx, "presetItemCreated",
		presetItemCreated{Status: "active"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if err := bundle.MetaEngine().Apply(ctx, "presetItemCreated",
		presetItemCreated{Status: "active"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	result, err := metaengine.ExecuteTyped[countInput, map[string]int64](
		ctx, bundle.MetaEngine(), countInput{},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if result["active"] != 2 {
		t.Errorf("active count: got %d, want 2", result["active"])
	}

	// The bundle Close should also close the metaengine store.
	if err := bundle.Close(); err != nil {
		t.Fatalf("bundle.Close: %v", err)
	}
}

type presetItemCreated struct {
	Status string
}
