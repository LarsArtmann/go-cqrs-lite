package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// FullFlow tests a complete CQRS pipeline:
//   - Command dispatch with middleware (recovery)
//   - Decider execution (load → decide → save → publish)
//   - Event store persistence
//   - Event bus delivery
//   - Projection live subscription
//   - Query dispatch with typed results
//   - Stream-based event loading
func TestFullFlow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryStore()
	defer store.Close()

	bus := eventtest.NewFakeBus()
	defer bus.Close()

	// --- Build command dispatcher with middleware ---
	cmdDispatcher := command.NewDispatcher()
	cmdDispatcher.Use(middleware.CommandRecovery())

	// --- Build query dispatcher ---
	qryDispatcher := query.NewDispatcher()

	// --- Set up decider for User aggregate ---
	userDecider := decider.Decider[UserState]{
		Initial: UserState{},
		Apply:   applyUserEvents,
	}

	userRepo, err := decider.NewRepository(store, bus, userDecider)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	// --- Register command handler ---
	cmdHandler := func(_ context.Context, cmd command.Command) error {
		c, ok := cmd.(*CreateUser)
		if !ok {
			return command.ErrTypeAssertion
		}

		return userRepo.Execute(
			ctx, c.StreamID(), "User",
			func(_ UserState, currentVersion event.Version) ([]event.Event, error) {
				evt, err := event.NewEvent(
					"UserCreated",
					c.StreamID(),
					"User",
					currentVersion.Add(1),
					[]byte(c.Name),
				)
				if err != nil {
					return nil, err
				}

				return []event.Event{evt}, nil
			},
		)
	}

	if err := cmdDispatcher.Register("CreateUser", cmdHandler); err != nil {
		t.Fatalf("register command: %v", err)
	}

	// --- Register query handler ---
	if err := query.RegisterTyped[*GetUser, UserState](
		qryDispatcher, "GetUser",
		func(_ context.Context, q *GetUser) (UserState, error) {
			events, err := store.Load(
				ctx,
				id.NewAggregateRef(id.StreamType("User"), q.StreamID),
			)
			if err != nil {
				return UserState{}, err
			}

			state := UserState{}

			for _, evt := range events {
				state, err = applyUserEvents(state, evt)
				if err != nil {
					return UserState{}, err
				}
			}

			return state, nil
		},
	); err != nil {
		t.Fatalf("register query: %v", err)
	}

	// --- Set up live event subscription ---
	var projectedNames []string

	_ = bus.SubscribeAll(func(_ context.Context, evt event.Event) error {
		if evt.Type() == "UserCreated" {
			projectedNames = append(projectedNames, string(evt.Payload()))
		}

		return nil
	})

	// --- Execute command ---
	aggID := id.NewAggregateID()

	createCmd, err := command.New("CreateUser", aggID)
	if err != nil {
		t.Fatalf("new command: %v", err)
	}

	// Wrap with name data (in real code, use a proper command struct)
	createCmdWithName := &CreateUser{
		BasicCommand: createCmd,
		Name:         "Alice",
	}

	if err := cmdDispatcher.Dispatch(ctx, createCmdWithName); err != nil {
		t.Fatalf("dispatch create user: %v", err)
	}

	// --- Verify events stored ---
	events, err := store.Load(ctx, id.NewAggregateRef(id.StreamType("User"), aggID))
	if err != nil {
		t.Fatalf("load events: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	eventtest.AssertEventType(t, events, 0, "UserCreated")

	// --- Verify projection received live event (poll for eventual consistency) ---
	waitForCondition(t, func() bool { return len(projectedNames) >= 1 }, 2*time.Second)

	if len(projectedNames) != 1 {
		t.Fatalf("expected 1 projected event, got %d", len(projectedNames))
	}

	if projectedNames[0] != "Alice" {
		t.Errorf("projected name = %q, want Alice", projectedNames[0])
	}

	// --- Execute query ---
	qry, err := query.New("GetUser")
	if err != nil {
		t.Fatalf("new query: %v", err)
	}

	getUserQuery := &GetUser{BasicQuery: qry, StreamID: aggID}

	result, err := query.DispatchTyped[UserState](ctx, qryDispatcher, getUserQuery)
	if err != nil {
		t.Fatalf("dispatch query: %v", err)
	}

	if result.Name != "Alice" {
		t.Errorf("query result name = %q, want Alice", result.Name)
	}

	// --- Verify stream loading works ---
	stream, err := store.LoadStream(ctx, id.NewAggregateRef(id.StreamType("User"), aggID))
	if err != nil {
		t.Fatalf("load stream: %v", err)
	}

	defer stream.Close()

	streamCount := 0

	for {
		_, err := stream.Next()
		if err != nil {
			break
		}

		streamCount++
	}

	if streamCount != 1 {
		t.Errorf("stream yielded %d events, want 1", streamCount)
	}
}

// --- Domain types ---

type UserState struct {
	Name string
}

type CreateUser struct {
	*command.BasicCommand

	Name string
}

type GetUser struct {
	*query.BasicQuery

	StreamID id.StreamID
}

// --- Decider functions ---

func applyUserEvents(state UserState, evt event.Event) (UserState, error) {
	switch evt.Type() {
	case "UserCreated":
		state.Name = string(evt.Payload())

		return state, nil
	default:
		return state, nil
	}
}
