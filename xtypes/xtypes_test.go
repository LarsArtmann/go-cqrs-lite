package xtypes

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/aggregate"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

type testRoot struct {
	*aggregate.Core
}

func (r *testRoot) Apply(_ event.Event) error { return nil }

var _ aggregate.Root = (*testRoot)(nil)

func TestEventBuilder(t *testing.T) {
	t.Parallel()

	t.Run("builds event with type safety", func(t *testing.T) {
		t.Parallel()

		aggregateID := id.NewAggregateID()
		aggregateType := event.AggregateType("TestAggregate")

		evt, err := NewEventBuilder(
			"TestCreated",
			aggregateID,
			aggregateType,
			1,
		).WithPayload([]byte(`{"name":"test"}`)).
			Build()
		if err != nil {
			t.Fatalf("NewEventBuilder() error = %v", err)
		}

		if evt.AggregateID() != aggregateID {
			t.Error("AggregateID mismatch")
		}
		if evt.Event().Type() != event.Type("TestCreated") {
			t.Error("EventType mismatch")
		}
	})

	t.Run("errors on empty aggregate ID", func(t *testing.T) {
		t.Parallel()

		var emptyID id.AggregateID

		_, err := NewEventBuilder(
			"TestCreated",
			emptyID,
			"TestAggregate",
			1,
		).Build()

		if err == nil {
			t.Error("Build() should error on empty aggregate ID")
		}
	})

	t.Run("errors on empty aggregate type", func(t *testing.T) {
		t.Parallel()

		aggregateID := id.NewAggregateID()

		_, err := NewEventBuilder(
			"TestCreated",
			aggregateID,
			"",
			1,
		).Build()

		if err == nil {
			t.Error("Build() should error on empty aggregate type")
		}
	})

	t.Run("errors on negative version", func(t *testing.T) {
		t.Parallel()

		aggregateID := id.NewAggregateID()

		_, err := NewEventBuilder(
			"TestCreated",
			aggregateID,
			"TestAggregate",
			-1,
		).Build()

		if err == nil {
			t.Error("Build() should error on negative version")
		}
	})

	t.Run("WithCorrelationID sets correlation ID", func(t *testing.T) {
		t.Parallel()

		aggregateID := id.NewAggregateID()
		correlationID := id.MustParseCorrelationID("corr-123")

		evt, err := NewEventBuilder("TestEvent", aggregateID, "TestAggregate", 1).
			WithCorrelationID(correlationID).
			Build()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if evt.Event().Metadata().CorrelationID != correlationID {
			t.Errorf(
				"expected correlation ID %s, got %s",
				correlationID,
				evt.Event().Metadata().CorrelationID,
			)
		}
	})

	t.Run("WithCausationID sets causation ID", func(t *testing.T) {
		t.Parallel()

		aggregateID := id.NewAggregateID()
		causationID := id.MustParseCausationID("cause-456")

		evt, err := NewEventBuilder("TestEvent", aggregateID, "TestAggregate", 1).
			WithCausationID(causationID).
			Build()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if evt.Event().Metadata().CausationID != causationID {
			t.Errorf(
				"expected causation ID %s, got %s",
				causationID,
				evt.Event().Metadata().CausationID,
			)
		}
	})

	t.Run("WithUserID sets user ID", func(t *testing.T) {
		t.Parallel()

		aggregateID := id.NewAggregateID()
		userID := id.MustParseUserID("user-789")

		evt, err := NewEventBuilder("TestEvent", aggregateID, "TestAggregate", 1).
			WithUserID(userID).
			Build()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if evt.Event().Metadata().UserID != userID {
			t.Errorf("expected user ID %s, got %s", userID, evt.Event().Metadata().UserID)
		}
	})

	t.Run("WithMetadata adds event options", func(t *testing.T) {
		t.Parallel()

		aggregateID := id.NewAggregateID()
		correlationID := id.MustParseCorrelationID("corr-meta")

		evt, err := NewEventBuilder("TestEvent", aggregateID, "TestAggregate", 1).
			WithMetadata(event.WithCorrelationID(correlationID)).
			Build()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if evt.Event().Metadata().CorrelationID != correlationID {
			t.Errorf(
				"expected correlation ID %s, got %s",
				correlationID,
				evt.Event().Metadata().CorrelationID,
			)
		}
	})

	t.Run("MustBuild panics on error", func(t *testing.T) {
		t.Parallel()

		defer func() {
			r := recover()
			if r == nil {
				t.Error("MustBuild() should panic on invalid input")
			}
		}()

		var emptyID id.AggregateID
		NewEventBuilder("TestCreated", emptyID, "TestAggregate", 1).MustBuild()
	})

	t.Run("MustBuild succeeds on valid input", func(t *testing.T) {
		t.Parallel()

		aggregateID := id.NewAggregateID()
		evt := NewEventBuilder("TestCreated", aggregateID, "TestAggregate", 1).
			WithPayload([]byte(`{"ok":true}`)).
			MustBuild()

		if evt == nil {
			t.Error("MustBuild() should return non-nil event")
		}
	})
}

func TestTypedEvent(t *testing.T) {
	t.Parallel()

	t.Run("Core returns underlying event.Core", func(t *testing.T) {
		t.Parallel()

		aggregateID := id.NewAggregateID()
		evt, err := NewEventBuilder("TestEvent", aggregateID, "TestAggregate", 1).Build()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		core := evt.Core()
		if core == nil {
			t.Error("Core() should return non-nil")
		}
		if core.Type() != event.Type("TestEvent") {
			t.Errorf("expected event type TestEvent, got %s", core.Type())
		}
	})

	t.Run("Event returns event.Event interface", func(t *testing.T) {
		t.Parallel()

		aggregateID := id.NewAggregateID()
		evt, err := NewEventBuilder("TestEvent", aggregateID, "TestAggregate", 1).Build()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		e := evt.Event()
		if e == nil {
			t.Error("Event() should return non-nil")
		}
		if e.AggregateID() != aggregateID.String() {
			t.Errorf("expected aggregate ID %s, got %s", aggregateID.String(), e.AggregateID())
		}
	})

	t.Run("AggregateID returns typed ID", func(t *testing.T) {
		t.Parallel()

		aggregateID := id.NewAggregateID()
		evt, err := NewEventBuilder("TestEvent", aggregateID, "TestAggregate", 1).Build()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if evt.AggregateID() != aggregateID {
			t.Errorf("expected aggregate ID %s, got %s", aggregateID, evt.AggregateID())
		}
	})
}

func TestTypedCommand(t *testing.T) {
	t.Parallel()

	t.Run("creates typed command", func(t *testing.T) {
		t.Parallel()

		aggregateID := id.NewAggregateID()
		cmd := NewTypedCommand("CreateTest", aggregateID)

		if cmd.Type() != "CreateTest" {
			t.Error("Type mismatch")
		}
		if cmd.AggregateID() != aggregateID {
			t.Error("AggregateID mismatch")
		}
	})

	t.Run("Command returns underlying command", func(t *testing.T) {
		t.Parallel()

		aggregateID := id.NewAggregateID()
		cmd := NewTypedCommand("CreateTest", aggregateID)

		c := cmd.Command()
		if c == nil {
			t.Error("Command() should return non-nil")
		}
		if c.Type() != "CreateTest" {
			t.Errorf("expected command type CreateTest, got %s", c.Type())
		}
	})
}

func TestTypedAggregate(t *testing.T) {
	t.Parallel()

	t.Run("creates typed aggregate", func(t *testing.T) {
		t.Parallel()
		aggregateID := id.NewAggregateID()
		agg := NewTypedAggregate(aggregateID, "TestAggregate")

		if agg.ID() != aggregateID {
			t.Error("ID mismatch")
		}
		if agg.Type() != "TestAggregate" {
			t.Error("Type mismatch")
		}
		if agg.Version() != 0 {
			t.Error("Version should start at 0")
		}
	})

	t.Run("Core returns underlying aggregate.Core", func(t *testing.T) {
		t.Parallel()
		aggregateID := id.NewAggregateID()
		agg := NewTypedAggregate(aggregateID, "TestAggregate")

		core := agg.Core()
		if core == nil {
			t.Error("Core() should return non-nil")
		}
		if core.ID() != aggregateID.String() {
			t.Errorf("expected core ID %s, got %s", aggregateID.String(), core.ID())
		}
	})

	t.Run("applies event", func(t *testing.T) {
		t.Parallel()
		aggregateID := id.NewAggregateID()
		agg := NewTypedAggregate(aggregateID, "TestAggregate")
		evt, _ := NewEventBuilder("TestCreated", aggregateID, "TestAggregate", 1).Build()
		agg.RecordEvent(context.Background(), evt)

		if agg.Version() != 1 {
			t.Errorf("Version should be 1, got %d", agg.Version())
		}
		if len(agg.UncommittedChanges()) != 1 {
			t.Error("Should have 1 uncommitted change")
		}
	})

	t.Run("loads from history", func(t *testing.T) {
		t.Parallel()
		aggregateID := id.NewAggregateID()
		agg := NewTypedAggregate(aggregateID, "TestAggregate")
		evt1, _ := NewEventBuilder("TestCreated", aggregateID, "TestAggregate", 1).Build()
		evt2, _ := NewEventBuilder("TestUpdated", aggregateID, "TestAggregate", 2).Build()

		err := agg.LoadFromHistory(&testRoot{agg.Core()}, []event.Event{evt1.Event(), evt2.Event()})
		if err != nil {
			t.Fatalf("LoadFromHistory() error = %v", err)
		}
		if agg.Version() != 2 {
			t.Errorf("Version should be 2, got %d", agg.Version())
		}
	})

	t.Run("marks changes as committed", func(t *testing.T) {
		t.Parallel()
		aggregateID := id.NewAggregateID()
		agg := NewTypedAggregate(aggregateID, "TestAggregate")
		evt, _ := NewEventBuilder("TestCreated", aggregateID, "TestAggregate", 1).Build()
		agg.RecordEvent(context.Background(), evt)

		if len(agg.UncommittedChanges()) != 1 {
			t.Fatalf("expected 1 change before commit")
		}

		agg.MarkChangesAsCommitted()

		if len(agg.UncommittedChanges()) != 0 {
			t.Errorf("expected 0 changes after commit, got %d", len(agg.UncommittedChanges()))
		}
		if agg.Version() != 1 {
			t.Errorf("version should remain 1 after commit, got %d", agg.Version())
		}
	})
}

func TestCommandID(t *testing.T) {
	t.Parallel()

	t.Run("NewCommandID generates non-empty ID", func(t *testing.T) {
		t.Parallel()

		cmdID := NewCommandID()
		if cmdID.IsEmpty() {
			t.Error("NewCommandID() should not return empty ID")
		}
	})

	t.Run("ParseCommandID parses valid string", func(t *testing.T) {
		t.Parallel()

		parsed, err := ParseCommandID("cmd-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if parsed.String() != "cmd-123" {
			t.Errorf("expected cmd-123, got %s", parsed.String())
		}
	})

	t.Run("ParseCommandID errors on empty string", func(t *testing.T) {
		t.Parallel()

		_, err := ParseCommandID("")
		if err == nil {
			t.Error("ParseCommandID() should error on empty string")
		}
	})
}
