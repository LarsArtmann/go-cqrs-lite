package stack_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	memory "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

type meTestKey string

type itemCreated struct {
	ID    meTestKey
	Title string
}

type meTestResult struct {
	ID    meTestKey
	Title string
}

func meQueryDecl() metaengine.QueryDecl[meTestKey, meTestResult] {
	return metaengine.Query[meTestKey, meTestResult](
		"me_test_items",
		metaengine.On(itemCreated{}, func(e itemCreated) (meTestKey, meTestResult) {
			return e.ID, meTestResult{ID: e.ID, Title: e.Title}
		}),
	)
}

func TestWithMetaEngine(t *testing.T) {
	t.Parallel()

	eng := metaengine.NewMemoryEngine()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, meQueryDecl())
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}

	// We need a minimal valid bundle: add an event store so validate() passes.
	memStore := memory.NewMemoryStore()

	bundle, err := stack.New(
		stack.WithEventStore(memStore),
		stack.WithMetaEngine(store),
	)
	if err != nil {
		t.Fatalf("stack.New: %v", err)
	}

	// Accessor returns the same store.
	if bundle.MetaEngine() == nil {
		t.Fatal("MetaEngine() returned nil, want non-nil")
	}

	if bundle.MetaEngine() != store {
		t.Fatal("MetaEngine() returned a different pointer than what was passed to WithMetaEngine")
	}

	// Close() should close the store (it implements io.Closer).
	if err := bundle.Close(); err != nil {
		t.Fatalf("bundle.Close: %v", err)
	}
}

func TestWithMetaEngine_Nil(t *testing.T) {
	t.Parallel()

	memStore := memory.NewMemoryStore()

	bundle, err := stack.New(
		stack.WithEventStore(memStore),
	)
	if err != nil {
		t.Fatalf("stack.New: %v", err)
	}

	if bundle.MetaEngine() != nil {
		t.Fatal("MetaEngine() should be nil when WithMetaEngine was not called")
	}

	if err := bundle.Close(); err != nil {
		t.Fatalf("bundle.Close: %v", err)
	}
}
