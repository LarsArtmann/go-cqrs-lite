// Package main — docs_compile_test.go verifies every API pattern shown in
// docs/getting-started.md compiles and runs correctly. This catches doc drift
// in CI: if an API changes, this test breaks.
package main

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memory "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// --- Types from docs/getting-started.md snippet 1: "Define Your Domain" ---

type (
	docUserCreated struct{ Name string }
	docUserState   struct{ Name string }
)

// --- Types from docs/getting-started.md snippet 4: "Commands with Typed Handlers" ---

type docCreateUser struct {
	*command.BasicCommand

	Name string
}

// TestDocsSnippet2_EventSourcingWithDecider verifies the event sourcing
// pattern from docs/getting-started.md section 2.
func TestDocsSnippet2_EventSourcingWithDecider(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryStore()
	bus := cqrswatermill.NewEventBus()

	d := decider.Decider[docUserState]{
		Initial: docUserState{},
		Apply: func(s docUserState, evt event.Event) (docUserState, error) {
			if evt.Type() != "user.created" {
				return s, nil
			}

			p, _ := event.DecodePayloadAuto[docUserCreated](evt)
			s.Name = p.Name

			return s, nil
		},
	}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	aggID := id.NewStreamID()

	err = repo.Execute(
		ctx,
		aggID,
		"User",
		func(_ docUserState, v event.Version) ([]event.Event, error) {
			return event.NewEvents(aggID, "User", v,
				[]event.Type{"user.created"}, []any{docUserCreated{Name: "Alice"}})
		},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	state, _, loadErr := repo.Load(ctx, aggID, "User")
	if loadErr != nil {
		t.Fatalf("Load: %v", loadErr)
	}

	if state.Name != "Alice" {
		t.Errorf("state.Name: got %q, want %q", state.Name, "Alice")
	}
}

// TestDocsSnippet3_BrandedIDs verifies the branded IDs pattern from
// docs/getting-started.md section 3.
func TestDocsSnippet3_BrandedIDs(t *testing.T) {
	t.Parallel()

	aggID := id.NewStreamID()
	eventID := id.NewEventID()

	if aggID.String() == "" {
		t.Error("aggID should not be empty")
	}

	if eventID.String() == "" {
		t.Error("eventID should not be empty")
	}

	type OrderID = id.Of[struct{}]
	orderID := id.New[OrderID]()

	if orderID.String() == "" {
		t.Error("orderID should not be empty")
	}
}

// TestDocsSnippet4_CommandsWithTypedHandlers verifies the command typed
// handler pattern from docs/getting-started.md section 4.
func TestDocsSnippet4_CommandsWithTypedHandlers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryStore()
	bus := cqrswatermill.NewEventBus()

	d := decider.Decider[docUserState]{
		Initial: docUserState{},
		Apply: func(s docUserState, evt event.Event) (docUserState, error) {
			if evt.Type() != "user.created" {
				return s, nil
			}

			p, _ := event.DecodePayloadAuto[docUserCreated](evt)
			s.Name = p.Name

			return s, nil
		},
	}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	aggID := id.NewStreamID()

	cmds := command.NewDispatcher()
	_ = command.RegisterTyped(cmds, "user.create",
		func(ctx context.Context, cmd *docCreateUser) error {
			return repo.Execute(
				ctx,
				cmd.StreamID(),
				"User",
				func(_ docUserState, v event.Version) ([]event.Event, error) {
					return event.NewEvents(cmd.StreamID(), "User", v,
						[]event.Type{"user.created"}, []any{docUserCreated{Name: cmd.Name}})
				},
			)
		})

	basic, basicErr := command.New("user.create", aggID)
	if basicErr != nil {
		t.Fatalf("command.New: %v", basicErr)
	}

	if err := cmds.Dispatch(ctx, &docCreateUser{BasicCommand: basic, Name: "Bob"}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	state, _, loadErr := repo.Load(ctx, aggID, "User")
	if loadErr != nil {
		t.Fatalf("Load: %v", loadErr)
	}

	if state.Name != "Bob" {
		t.Errorf("state.Name: got %q, want %q", state.Name, "Bob")
	}
}
