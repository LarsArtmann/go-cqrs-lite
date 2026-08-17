package scheduling_test

import (
	"context"
	"encoding/json/v2"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

// TestTimer_ActorDeliveredToDispatch proves the actor recorded on a timer
// reaches the dispatch callback — the propagation point where consumers stamp
// command.WithActor for timer-initiated commands.
func TestTimer_ActorDeliveredToDispatch(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	store := scheduling.NewMemoryTimerStore[testPayload]()

	var gotActor string

	fired := make(chan struct{})

	sched := scheduling.New[testPayload](
		store,
		func(_ context.Context, timer scheduling.Timer[testPayload]) error {
			gotActor = timer.Actor
			close(fired)

			return nil
		},
	)

	go sched.Start(ctx)

	err := store.Schedule(ctx, scheduling.Timer[testPayload]{
		ID:      "actor-dispatch",
		FireAt:  time.Now().Add(10 * time.Millisecond),
		Payload: testPayload{Action: "cancel"},
		Actor:   "user:01HK1540X0841Y0A6BSX1VKR99",
	})
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	select {
	case <-fired:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for timer dispatch")
	}

	if want := "user:01HK1540X0841Y0A6BSX1VKR99"; gotActor != want {
		t.Errorf("dispatch actor = %q, want %q", gotActor, want)
	}
}

// TestTimer_JSONOmitsZeroActor pins the wire shape: the actor field is absent
// when unset (omitzero) and carries the self-describing "kind:raw" form when
// set — the same JSON every TimerStore serializes.
func TestTimer_JSONOmitsZeroActor(t *testing.T) {
	t.Parallel()

	withoutActor, err := json.Marshal(scheduling.Timer[string]{ID: "t1", Payload: "p"})
	if err != nil {
		t.Fatalf("marshal without actor: %v", err)
	}

	if got, want := string(
		withoutActor,
	), `{"id":"t1","fireAt":"0001-01-01T00:00:00Z","payload":"p"}`; got != want {
		t.Errorf("zero-actor JSON = %s, want %s", got, want)
	}

	withActor, err := json.Marshal(
		scheduling.Timer[string]{ID: "t1", Payload: "p", Actor: "system:scheduler"},
	)
	if err != nil {
		t.Fatalf("marshal with actor: %v", err)
	}

	var decoded scheduling.Timer[string]

	if err := json.Unmarshal(withActor, &decoded); err != nil {
		t.Fatalf("unmarshal with actor: %v", err)
	}

	if decoded.Actor != "system:scheduler" {
		t.Errorf("actor lost through JSON round-trip: got %q", decoded.Actor)
	}
}

type testPayload struct {
	Action string `json:"action"`
	Amount int    `json:"amount"`
}
