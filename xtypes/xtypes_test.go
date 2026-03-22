package xtypes

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

type testAggregateBrand struct{}

func TestEventBuilder(t *testing.T) {
	t.Parallel()

	t.Run("builds event with type safety", func(t *testing.T) {
		aggregateID := id.New[id.Of[testAggregateBrand]]()
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
		if evt.Event().Type() != event.EventType("TestCreated") {
			t.Error("EventType mismatch")
		}
	})

	t.Run("errors on empty aggregate ID", func(t *testing.T) {
		var emptyID id.Of[testAggregateBrand]

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

	t.Run("errors on negative version", func(t *testing.T) {
		aggregateID := id.New[id.Of[testAggregateBrand]]()

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
}

func TestTypedCommand(t *testing.T) {
	t.Parallel()

	t.Run("creates typed command", func(t *testing.T) {
		aggregateID := id.New[id.Of[testAggregateBrand]]()
		cmd := NewTypedCommand("CreateTest", aggregateID)

		if cmd.Type() != "CreateTest" {
			t.Error("Type mismatch")
		}
		if cmd.AggregateID() != aggregateID {
			t.Error("AggregateID mismatch")
		}
	})
}

func TestTypedAggregate(t *testing.T) {
	t.Parallel()

	t.Run("creates typed aggregate", func(t *testing.T) {
		aggregateID := id.New[id.Of[testAggregateBrand]]()
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

	t.Run("applies event", func(t *testing.T) {
		aggregateID := id.New[id.Of[testAggregateBrand]]()
		agg := NewTypedAggregate(aggregateID, "TestAggregate")

		evt, _ := NewEventBuilder(
			"TestCreated",
			aggregateID,
			"TestAggregate",
			1,
		).Build()

		agg.ApplyEvent(context.Background(), evt)

		if agg.Version() != 1 {
			t.Errorf("Version should be 1, got %d", agg.Version())
		}
		if len(agg.UncommittedChanges()) != 1 {
			t.Error("Should have 1 uncommitted change")
		}
	})

	t.Run("loads from history", func(t *testing.T) {
		aggregateID := id.New[id.Of[testAggregateBrand]]()
		agg := NewTypedAggregate(aggregateID, "TestAggregate")

		evt1, _ := NewEventBuilder(
			"TestCreated",
			aggregateID,
			"TestAggregate",
			1,
		).Build()

		evt2, _ := NewEventBuilder(
			"TestUpdated",
			aggregateID,
			"TestAggregate",
			2,
		).Build()

		err := agg.LoadFromHistory([]event.Event{evt1.Event(), evt2.Event()})
		if err != nil {
			t.Fatalf("LoadFromHistory() error = %v", err)
		}

		if agg.Version() != 2 {
			t.Errorf("Version should be 2, got %d", agg.Version())
		}
	})
}
