package deriver

import (
	"context"
	"errors"
	"testing"

	cqrscommand "github.com/larsartmann/go-cqrs-lite/command/v4"
	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func testEvent(t *testing.T, eventType string) cqrsevent.Event {
	t.Helper()

	aggID := id.NewAggregateID()
	evt, err := cqrsevent.NewEvent(
		cqrsevent.Type(eventType),
		aggID,
		"TestAggregate",
		cqrsevent.Version(1),
		[]byte("{}"),
	)
	if err != nil {
		t.Fatalf("new event: %v", err)
	}

	return evt
}

func TestDeriver_Then(t *testing.T) {
	t.Parallel()

	d1 := Deriver(func(_ context.Context, _ cqrsevent.Event) ([]cqrscommand.Command, error) {
		cmd, _ := cqrscommand.New("cmd.a", id.NewAggregateID())

		return []cqrscommand.Command{cmd}, nil
	})

	d2 := Deriver(func(_ context.Context, _ cqrsevent.Event) ([]cqrscommand.Command, error) {
		cmd, _ := cqrscommand.New("cmd.b", id.NewAggregateID())

		return []cqrscommand.Command{cmd}, nil
	})

	combined := d1.Then(d2)

	evt := testEvent(t, "test.event")
	cmds, err := combined(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}

	if string(cmds[0].Type()) != "cmd.a" {
		t.Fatalf("first command type = %q, want cmd.a", cmds[0].Type())
	}

	if string(cmds[1].Type()) != "cmd.b" {
		t.Fatalf("second command type = %q, want cmd.b", cmds[1].Type())
	}
}

func TestDeriver_ThenErrorPropagation(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("d1 failed")

	d1 := Deriver(func(_ context.Context, _ cqrsevent.Event) ([]cqrscommand.Command, error) {
		return nil, wantErr
	})

	called := false

	d2 := Deriver(func(_ context.Context, _ cqrsevent.Event) ([]cqrscommand.Command, error) {
		called = true

		return nil, nil
	})

	combined := d1.Then(d2)
	_, err := combined(context.Background(), testEvent(t, "test"))

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wantErr, got %v", err)
	}

	if called {
		t.Fatal("d2 should not be called when d1 errors")
	}
}

func TestDeriver_Filter(t *testing.T) {
	t.Parallel()

	called := false

	d := Deriver(func(_ context.Context, _ cqrsevent.Event) ([]cqrscommand.Command, error) {
		called = true
		cmd, _ := cqrscommand.New("cmd.x", id.NewAggregateID())

		return []cqrscommand.Command{cmd}, nil
	}).Filter("user.created", "user.updated")

	// Matching event type.
	called = false
	cmds, err := d(context.Background(), testEvent(t, "user.created"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Fatal("deriver should be called for matching event type")
	}

	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}

	// Non-matching event type.
	called = false
	cmds, err = d(context.Background(), testEvent(t, "user.deleted"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if called {
		t.Fatal("deriver should NOT be called for non-matching event type")
	}

	if cmds != nil {
		t.Fatalf("expected nil commands for filtered event, got %d", len(cmds))
	}
}

func TestDeriver_AsHandler(t *testing.T) {
	t.Parallel()

	dispatched := make([]string, 0)

	dispatcher := cqrscommand.NewDispatcher()

	_ = dispatcher.Register(
		"cmd.derived",
		cqrscommand.Handler(func(_ context.Context, _ cqrscommand.Command) error {
			dispatched = append(dispatched, "cmd.derived")

			return nil
		}),
	)

	d := Deriver(func(_ context.Context, _ cqrsevent.Event) ([]cqrscommand.Command, error) {
		cmd, _ := cqrscommand.New("cmd.derived", id.NewAggregateID())

		return []cqrscommand.Command{cmd}, nil
	})

	handler := d.AsHandler(dispatcher)

	err := handler(context.Background(), testEvent(t, "test.event"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dispatched) != 1 {
		t.Fatalf("expected 1 dispatched command, got %d", len(dispatched))
	}
}

func TestDeriver_AsHandler_NilDispatcher(t *testing.T) {
	t.Parallel()

	d := Noop()
	handler := d.AsHandler(nil)

	err := handler(context.Background(), testEvent(t, "test.event"))
	if !errors.Is(err, ErrNilDispatcher) {
		t.Fatalf("expected ErrNilDispatcher, got %v", err)
	}
}

func TestDeriver_Noop(t *testing.T) {
	t.Parallel()

	cmds, err := Noop()(context.Background(), testEvent(t, "test.event"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cmds != nil {
		t.Fatalf("expected nil commands, got %d", len(cmds))
	}
}

func TestDeriver_AsHandlerErrorPropagation(t *testing.T) {
	t.Parallel()

	dispatcher := cqrscommand.NewDispatcher()

	wantErr := errors.New("derive failed")

	d := Deriver(func(_ context.Context, _ cqrsevent.Event) ([]cqrscommand.Command, error) {
		return nil, wantErr
	})

	handler := d.AsHandler(dispatcher)

	err := handler(context.Background(), testEvent(t, "test.event"))
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wantErr, got %v", err)
	}
}

func TestDeriver_Idempotent_DeterministicIDs(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()

	d := Deriver(func(_ context.Context, _ cqrsevent.Event) ([]cqrscommand.Command, error) {
		cmd1, _ := cqrscommand.New("cmd.first", aggID)
		cmd2, _ := cqrscommand.New("cmd.second", aggID)

		return []cqrscommand.Command{cmd1, cmd2}, nil
	}).Idempotent()

	evt := testEvent(t, "source.event")

	// Run twice — the same event must yield the same command IDs.
	first, err := d(context.Background(), evt)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	second, err := d(context.Background(), evt)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	if len(first) != 2 || len(second) != 2 {
		t.Fatalf("expected 2 commands each, got %d and %d", len(first), len(second))
	}

	for i := range first {
		if first[i].ID() != second[i].ID() {
			t.Errorf("command[%d]: first ID %v != second ID %v (not deterministic)",
				i, first[i].ID(), second[i].ID())
		}
	}
}

func TestDeriver_Idempotent_DifferentEvents(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()

	d := Deriver(func(_ context.Context, _ cqrsevent.Event) ([]cqrscommand.Command, error) {
		cmd, _ := cqrscommand.New("cmd.derived", aggID)

		return []cqrscommand.Command{cmd}, nil
	}).Idempotent()

	cmds1, _ := d(context.Background(), testEvent(t, "source.one"))
	cmds2, _ := d(context.Background(), testEvent(t, "source.two"))

	if cmds1[0].ID() == cmds2[0].ID() {
		t.Error("commands derived from different events should have different IDs")
	}
}

func TestDeriver_Idempotent_SourceEventMetadata(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()

	d := Deriver(func(_ context.Context, _ cqrsevent.Event) ([]cqrscommand.Command, error) {
		cmd, _ := cqrscommand.New("cmd.derived", aggID)

		return []cqrscommand.Command{cmd}, nil
	}).Idempotent()

	evt := testEvent(t, "source.event")
	cmds, err := d(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bc, ok := cmds[0].(*cqrscommand.BasicCommand)
	if !ok {
		t.Fatal("expected *BasicCommand")
	}

	md := bc.Metadata()
	sourceID, ok := md.Custom[SourceEventIDKey]
	if !ok {
		t.Fatal("expected source_event_id in command metadata")
	}

	if sourceID != evt.ID().String() {
		t.Errorf("source_event_id = %q, want %q", sourceID, evt.ID().String())
	}
}

func TestDeriver_Idempotent_ErrorPropagation(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("derive failed")

	d := Deriver(func(_ context.Context, _ cqrsevent.Event) ([]cqrscommand.Command, error) {
		return nil, wantErr
	}).Idempotent()

	_, err := d(context.Background(), testEvent(t, "test.event"))
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wantErr, got %v", err)
	}
}

func TestDeriver_Idempotent_NoCommands(t *testing.T) {
	t.Parallel()

	d := Noop().Idempotent()

	cmds, err := d(context.Background(), testEvent(t, "test.event"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cmds != nil {
		t.Fatalf("expected nil commands, got %d", len(cmds))
	}
}

func TestDeriver_ChainedComposition(t *testing.T) {
	t.Parallel()

	dispatched := make([]string, 0)

	dispatcher := cqrscommand.NewDispatcher()

	for _, cmdType := range []string{"welcome", "crm.sync", "analytics"} {
		ct := cmdType // capture
		_ = dispatcher.Register(
			cqrscommand.Type(ct),
			cqrscommand.Handler(func(_ context.Context, _ cqrscommand.Command) error {
				dispatched = append(dispatched, ct)

				return nil
			}),
		)
	}

	mkDeriver := func(cmdType string) Deriver {
		return Deriver(func(_ context.Context, _ cqrsevent.Event) ([]cqrscommand.Command, error) {
			cmd, _ := cqrscommand.New(cqrscommand.Type(cmdType), id.NewAggregateID())

			return []cqrscommand.Command{cmd}, nil
		})
	}

	// Chain three derivers with Filter.
	d := mkDeriver("welcome").
		Then(mkDeriver("crm.sync")).
		Then(mkDeriver("analytics")).
		Filter("user.created")

	handler := d.AsHandler(dispatcher)

	// Matching event.
	err := handler(context.Background(), testEvent(t, "user.created"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dispatched) != 3 {
		t.Fatalf("expected 3 dispatched commands, got %d: %v", len(dispatched), dispatched)
	}

	// Non-matching event — should produce nothing.
	dispatched = dispatched[:0]
	err = handler(context.Background(), testEvent(t, "user.deleted"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dispatched) != 0 {
		t.Fatalf("expected 0 dispatched commands for filtered event, got %d", len(dispatched))
	}
}
