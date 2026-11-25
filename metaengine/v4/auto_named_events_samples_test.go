package metaengine_test

import (
	"context"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

type (
	ptrCreated struct {
		ID   string
		Name string
	}
	ptrUpdated struct {
		ID   string
		Name string
	}
	ptrDeleted struct {
		ID string
	}
	ptrView    struct{ ID, Name string }
	ptrFindOne struct{ ID string }
)

// Pointer samples are accepted by the suffix scan, so the folds must be built
// from the dereferenced struct type.
func TestAutoCRUDByNamedEvents_PointerSamples(t *testing.T) {
	t.Parallel()

	folds, err := metaengine.AutoCRUDByNamedEvents[ptrView]("ID",
		metaengine.NamedEvent("ptr.created", &ptrCreated{}),
		metaengine.NamedEvent("ptr.updated", &ptrUpdated{}),
		metaengine.NamedEvent("ptr.deleted", &ptrDeleted{}),
	)
	if err != nil {
		t.Fatalf("AutoCRUDByNamedEvents: %v", err)
	}

	if len(folds) != 3 {
		t.Fatalf("want 3 folds, got %d", len(folds))
	}

	query := metaengine.Query[ptrFindOne, ptrView]("ptr_view", folds[0], folds[1], folds[2])

	store, err := metaengine.PlanFromMemory(query)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	defer store.Close()

	ctx := context.Background()

	if err := store.Apply(ctx, "ptr.created", ptrCreated{ID: "p1", Name: "old"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if err := store.Apply(ctx, "ptr.updated", ptrUpdated{ID: "p1", Name: "new"}); err != nil {
		t.Fatalf("Apply update: %v", err)
	}

	got, err := metaengine.ExecuteTyped[ptrFindOne, ptrView](ctx, store, ptrFindOne{ID: "p1"})
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if got.Name != "new" {
		t.Errorf("Name=%q, want %q", got.Name, "new")
	}

	if err := store.Apply(ctx, "ptr.deleted", ptrDeleted{ID: "p1"}); err != nil {
		t.Fatalf("Apply delete: %v", err)
	}
}

func TestAutoCRUD_NilSampleReturnsError(t *testing.T) {
	t.Parallel()

	_, namedErr := metaengine.AutoCRUDByNamedEvents[ptrView]("ID",
		metaengine.NamedEvent("ptr.created", ptrCreated{}),
		metaengine.NamedEvent("ptr.updated", nil),
	)
	if namedErr == nil || !strings.Contains(namedErr.Error(), `"ptr.updated"`) {
		t.Errorf("AutoCRUDByNamedEvents: want an error naming the event type, got %v", namedErr)
	}

	_, convErr := metaengine.AutoCRUDByConvention[ptrView]("ID", ptrCreated{}, nil)
	if convErr == nil || !strings.Contains(convErr.Error(), "nil") {
		t.Errorf("AutoCRUDByConvention: want a nil-sample error, got %v", convErr)
	}
}
