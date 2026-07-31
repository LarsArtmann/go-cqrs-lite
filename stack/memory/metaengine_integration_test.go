package memory_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	stackmemory "github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
)

// meItemKey is a test key type for metaengine integration.
type meItemKey string

func (k meItemKey) String() string { return string(k) }

type meItemView struct {
	ID    meItemKey
	Title string
}

type meItemCreated struct {
	ID    meItemKey
	Title string
}

// TestNew_WithMetaEngine verifies that memory.New(stack.WithMetaEngine(store))
// produces a fully wired bundle with both the default capabilities AND the
// metaengine store attached. This is the variadic-options path added to
// support benchkit and cqrs-bench.
func TestNew_WithMetaEngine(t *testing.T) {
	t.Parallel()

	eng := metaengine.NewMemoryEngine()

	store, err := metaengine.Plan([]metaengine.Engine{eng},
		metaengine.Query[meItemKey, meItemView](
			"items",
			metaengine.On(meItemCreated{}, func(e meItemCreated) (meItemKey, meItemView) {
				return e.ID, meItemView(e)
			}),
		),
	)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}

	bundle, err := stackmemory.New(stack.WithMetaEngine(store))
	if err != nil {
		t.Fatalf("memory.New(WithMetaEngine): %v", err)
	}

	defer func() { _ = bundle.Close() }()

	// MetaEngine must be set.
	if bundle.MetaEngine() == nil {
		t.Fatal("MetaEngine() is nil after memory.New(WithMetaEngine(store))")
	}

	if bundle.MetaEngine() != store {
		t.Fatal("MetaEngine() returned a different pointer than what was passed")
	}

	// All default capabilities must still be wired.
	checks := []struct {
		name string
		ok   bool
	}{
		{"EventSink", bundle.EventSink != nil},
		{"EventSource", bundle.EventSource != nil},
		{"Publisher", bundle.Publisher != nil},
		{"CommandSink", bundle.CommandSink != nil},
		{"QuerySink", bundle.QuerySink != nil},
		{"SnapshotStore", bundle.SnapshotStore != nil},
		{"CheckpointStore", bundle.CheckpointStore != nil},
		{"ReadModels", bundle.ReadModels != nil},
	}

	for _, c := range checks {
		if !c.ok {
			t.Errorf("capability %s not set after WithMetaEngine", c.name)
		}
	}
}

// TestNew_WithMetaEngine_NilExtraOptions verifies that memory.New() without
// extra options still works (backward compatibility).
func TestNew_WithMetaEngine_NilExtraOptions(t *testing.T) {
	t.Parallel()

	bundle, err := stackmemory.New()
	if err != nil {
		t.Fatalf("memory.New(): %v", err)
	}

	defer func() { _ = bundle.Close() }()

	// MetaEngine should be nil when not provided.
	if bundle.MetaEngine() != nil {
		t.Error("MetaEngine() should be nil when WithMetaEngine was not passed")
	}
}
