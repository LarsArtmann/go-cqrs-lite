package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/middleware/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
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
	defer store.Close() //nolint:errcheck // test helper

	bus := eventtest.NewFakeBus()
	defer bus.Close() //nolint:errcheck // test helper

	// --- Build command dispatcher with middleware ---
	cmdDispatcher := command.NewDispatcher()
	cmdDispatcher.Use(middleware.CommandRecovery())

	// --- Build query dispatcher ---
	qryDispatcher := query.NewDispatcher()

	// --- Set up decider for User aggregate ---
	userDecider := decider.Decider[UserState]{
		Initial: UserState{},
		Fold:    foldUserEvents,
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
			ctx, c.AggregateID(), "User",
			func(_ UserState, currentVersion event.Version) ([]event.Event, error) {
				evt, err := event.NewEvent(
					"UserCreated",
					c.AggregateID(),
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
				event.NewAggregateRef(event.AggregateType("User"), q.AggregateID),
			)
			if err != nil {
				return UserState{}, err
			}

			state := UserState{}

			for _, evt := range events {
				state, err = foldUserEvents(state, evt)
				if err != nil {
					return UserState{}, err
				}
			}

			return state, nil
		},
	); err != nil {
		t.Fatalf("register query: %v", err)
	}

	// --- Set up projection with live subscription ---
	checkpoints := memory.NewMemoryCheckpointStore()
	projRunner, err := projection.NewRunner(nil, bus, checkpoints)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	var projectedNames []string

	if err := projRunner.Register(event.NewProjection(
		"user-names",
		func(_ context.Context, evt event.Event) error {
			if evt.Type() == "UserCreated" {
				projectedNames = append(projectedNames, string(evt.Payload()))
			}

			return nil
		},
		[]event.Type{"UserCreated"},
	)); err != nil {
		t.Fatalf("register projection: %v", err)
	}

	// Start projection runner in background
	projCtx, projCancel := context.WithCancel(ctx)
	defer projCancel()

	go func() {
		if runErr := projRunner.Run(projCtx); runErr != nil {
			t.Logf("projection runner: %v", runErr)
		}
	}()

	// Small delay to let subscription set up — the runner subscribes in Run(),
	// which starts in a goroutine above. For in-process buses this is near-instant.
	time.Sleep(10 * time.Millisecond)

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
	events, err := store.Load(ctx, event.NewAggregateRef(event.AggregateType("User"), aggID))
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

	getUserQuery := &GetUser{BasicQuery: qry, AggregateID: aggID}

	result, err := query.DispatchTyped[UserState](ctx, qryDispatcher, getUserQuery)
	if err != nil {
		t.Fatalf("dispatch query: %v", err)
	}

	if result.Name != "Alice" {
		t.Errorf("query result name = %q, want Alice", result.Name)
	}

	// --- Verify stream loading works ---
	stream, err := store.LoadStream(ctx, event.NewAggregateRef(event.AggregateType("User"), aggID))
	if err != nil {
		t.Fatalf("load stream: %v", err)
	}

	defer stream.Close() //nolint:errcheck // test helper

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

	AggregateID id.AggregateID
}

// --- Decider functions ---

func foldUserEvents(state UserState, evt event.Event) (UserState, error) {
	switch evt.Type() {
	case "UserCreated":
		state.Name = string(evt.Payload())

		return state, nil
	default:
		return state, nil
	}
}
