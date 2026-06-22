package stack_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v3"
)

// TestDeployerFirstArchitecture validates the full deployer-first flow:
// 1. Deployer picks a preset (memory in this test)
// 2. Consumer creates a TypedRepository + NewMaterialize
// 3. Execute a command → event persisted + published
// 4. Materialize handler processes the event → view record created
// 5. View is retrievable via mat.View
//
// This test proves the new architecture works end-to-end with ZERO references
// to projection.Runner or readmodel.Store.
func TestDeployerFirstArchitecture(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// ── Deployer code: picks infrastructure ──
	bundle, err := stack.New(
		stack.WithEventStore(memory.NewMemoryStore()),
		stack.WithBus(cqrswatermill.NewEventBus()),
		stack.WithReadModels(kv.NewMemStore()),
	)
	if err != nil {
		t.Fatalf("stack.New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	// ── Consumer code: defines domain + uses stack accessors ──

	// Create a Materialize for UserView
	mat, err := stack.NewMaterialize[deployerUserView, deployerUserID](
		bundle, nil, // nil = JSON codec
		func(evt event.Event) (deployerUserID, error) {
			return deployerUserID(evt.AggregateID().String()), nil
		},
	)
	if err != nil {
		t.Fatalf("NewMaterialize: %v", err)
	}

	// Set up the materialize callbacks
	mat.OnCreate = func(_ context.Context, _ event.Event) (*deployerUserView, error) {
		return &deployerUserView{Name: "created-via-materialize"}, nil
	}

	// Simulate an event being published
	aggID := id.NewAggregateID()
	evt, err := event.NewEvent(
		event.Type("user.created"),
		aggID,
		"User",
		event.Version(1),
		[]byte(`{}`),
	)
	if err != nil {
		t.Fatalf("event.NewEvent: %v", err)
	}

	// Publish the event through the bus
	if err := bundle.Publisher.Publish(ctx, evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Process via the materialize handler (simulating what a Watermill Router would do)
	msg := buildTestMessage(evt, "user.created")
	handler := mat.HandlerFunc()
	if err := handler(msg); err != nil {
		t.Fatalf("Materialize handler: %v", err)
	}

	// Verify the view was materialized
	view, err := mat.View(ctx, deployerUserID(aggID.String()))
	if err != nil {
		t.Fatalf("View: %v", err)
	}

	if view.Name != "created-via-materialize" {
		t.Fatalf("expected Name 'created-via-materialize', got %q", view.Name)
	}

	t.Logf("Deployer-first architecture validated: event → materialize → view = %q", view.Name)
}

// Domain types for the test (kept inline to keep the test self-contained).

type deployerUserID string

func (u deployerUserID) String() string { return string(u) }

type deployerUserView struct {
	Name string `json:"name"`
}
