package metaengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/metaengine/v4"
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
		metaengine.OnRecord(
			itemCreated{},
			func(_ record.Record, e itemCreated) (meTestKey, meTestResult) {
				return e.ID, meTestResult(e)
			},
		),
	)
}

// TestWithStore verifies the typed registration round-trip: WithStore wires
// the concrete *metaengine.Store onto the Bundle, Store recovers it, and
// Bundle.Close closes it.
func TestWithStore(t *testing.T) {
	t.Parallel()

	eng := metaengine.NewMemoryEngine()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, meQueryDecl())
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}

	memStore := memory.NewMemoryStore()

	bundle, err := stack.New(
		stack.WithEventStore(memStore),
		metaengine.WithStore(store),
	)
	if err != nil {
		t.Fatalf("stack.New: %v", err)
	}

	if got := metaengine.Store(bundle); got != store {
		t.Fatal("Store(bundle) returned a different pointer than what WithStore registered")
	}
	if bundle.MetaEngine() == nil {
		t.Fatal("deprecated MetaEngine() accessor returned nil, want non-nil")
	}

	if err := bundle.Close(); err != nil {
		t.Fatalf("bundle.Close: %v", err)
	}
}

// TestWithStore_Nil verifies Store returns nil when nothing was registered.
func TestWithStore_Nil(t *testing.T) {
	t.Parallel()

	memStore := memory.NewMemoryStore()

	bundle, err := stack.New(
		stack.WithEventStore(memStore),
	)
	if err != nil {
		t.Fatalf("stack.New: %v", err)
	}

	if got := metaengine.Store(bundle); got != nil {
		t.Fatal("Store(bundle) should be nil when WithStore was not called")
	}
	if bundle.MetaEngine() != nil {
		t.Fatal("MetaEngine() should be nil when WithStore was not called")
	}

	if err := bundle.Close(); err != nil {
		t.Fatalf("bundle.Close: %v", err)
	}
}

// TestDeprecatedSeamStillAcceptsConcreteStore pins source compatibility: a
// *metaengine.Store passed to the deprecated stack.WithMetaEngine (now typed
// as the MetaEngineStore seam) still wires and stays recoverable.
func TestDeprecatedSeamStillAcceptsConcreteStore(t *testing.T) {
	t.Parallel()

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, meQueryDecl())
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}

	bundle, err := stack.New(
		stack.WithEventStore(memory.NewMemoryStore()),
		stack.WithMetaEngine(store),
	)
	if err != nil {
		t.Fatalf("stack.New: %v", err)
	}

	if got := metaengine.Store(bundle); got != store {
		t.Fatal("Store(bundle) did not recover the store registered via the deprecated seam")
	}

	if err := bundle.Close(); err != nil {
		t.Fatalf("bundle.Close: %v", err)
	}
}
