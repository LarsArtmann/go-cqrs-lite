package metaengine_test

import (
	"context"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

type (
	reorderCreated struct {
		OrderID   orderLineID
		ProductID productID
		Quantity  int
	}
	// Key fields in a different order than the Created event.
	reorderUpdated struct {
		ProductID productID
		Quantity  int
		OrderID   orderLineID
	}
	// Key fields missing entirely.
	reorderBrokenDeleted struct {
		OrderID orderLineID
	}
)

// The composite key must be resolved by field name per event type, not by
// the Created event's field offsets.
func TestInfer_CompositeKeyUpdateReorderedFields(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[getOrderLine, orderLineView]("reorder_composite",
		metaengine.Infer(reorderCreated{}, reorderUpdated{}),
	)

	store, err := metaengine.PlanFromMemory(q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	defer store.Close()

	ctx := context.Background()

	if err := store.Apply(ctx, "reorderCreated", reorderCreated{
		OrderID: "o", ProductID: "p", Quantity: 1,
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if err := store.Apply(ctx, "reorderUpdated", reorderUpdated{
		ProductID: "p", Quantity: 7, OrderID: "o",
	}); err != nil {
		t.Fatalf("Apply update: %v", err)
	}

	got, err := metaengine.ExecuteTyped[getOrderLine, orderLineView](
		ctx, store, getOrderLine{OrderID: "o", ProductID: "p"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if got.Quantity != 7 {
		t.Errorf("update hit the wrong row: Quantity=%d, want 7", got.Quantity)
	}
}

// An update/delete event without a key field is rejected when the query is
// planned, instead of panicking on the first applied event.
func TestInfer_CompositeKeyMissingFieldRejectedAtPlan(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[getOrderLine, orderLineView]("broken_composite",
		metaengine.Infer(reorderCreated{}, reorderBrokenDeleted{}),
	)

	_, err := metaengine.PlanFromMemory(q)
	if err == nil || !strings.Contains(err.Error(), "reorderBrokenDeleted") {
		t.Fatalf("want a plan error naming the event type, got %v", err)
	}
}
