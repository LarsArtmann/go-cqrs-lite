package testhelpers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestEventHelpers_NewTestEvent(t *testing.T) {
	t.Parallel()

	evt, err := NewTestEvent()
	if err != nil {
		t.Fatalf("NewTestEvent: %v", err)
	}

	if evt.Type() != "test.evt" {
		t.Errorf("Type = %q, want test.evt", evt.Type())
	}
}

func TestEventHelpers_NewEvent(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	evt := NewEvent(t, "user.created", aggID, "User", 1, []byte(`{}`))

	if evt.AggregateID() != aggID {
		t.Error("AggregateID mismatch")
	}
}

func TestEventHelpers_NewEventOpts(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	corrID := id.NewCorrelationID()

	evt := NewEventOpts(t, "user.created", aggID, "User", 1, nil,
		event.WithCorrelationID(corrID),
	)

	if evt.AggregateID() != aggID {
		t.Error("AggregateID mismatch")
	}

	if evt.Metadata().CorrelationID != corrID {
		t.Error("CorrelationID mismatch")
	}
}

func TestEventHelpers_MakeEvent(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	evt, err := MakeEvent("user.created", aggID, "User", 1, nil)
	if err != nil {
		t.Fatalf("MakeEvent: %v", err)
	}

	if evt.AggregateID() != aggID {
		t.Error("AggregateID mismatch")
	}
}

func TestEventHelpers_QuickEvent(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	evt := QuickEvent("user.created", aggID, "User", 1, nil)

	if evt.Type() != "user.created" {
		t.Errorf("Type = %q, want user.created", evt.Type())
	}
}

func TestEventHelpers_QuickEventOpts(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	corrID := id.NewCorrelationID()

	evt := QuickEventOpts(
		"user.created",
		aggID,
		"User",
		1,
		nil,
		event.WithCorrelationID(corrID),
	)

	if evt.Type() != "user.created" {
		t.Errorf("Type = %q, want user.created", evt.Type())
	}

	if evt.Metadata().CorrelationID != corrID {
		t.Error("CorrelationID mismatch")
	}
}

func TestFakeMetrics_Observe(t *testing.T) {
	t.Parallel()

	m := &FakeMetrics{}
	m.Observe("command.create", 100*time.Millisecond)

	AssertMetricRecord(t, m, "command.create")
}

func TestAssertCallOrder(t *testing.T) {
	t.Parallel()

	AssertCallOrder(t, []string{"a", "b", "c"}, []string{"a", "b", "c"})
}

func TestAssertLen(t *testing.T) {
	t.Parallel()

	AssertLen(t, "items", []string{"a", "b"}, 2)
}

func TestAssertLenFatal(t *testing.T) {
	t.Parallel()

	AssertLenFatal(t, "items", []int{1}, 1)
}

func TestAssertNoError(t *testing.T) {
	t.Parallel()

	AssertNoError(t, nil, "should be nil")
}

func TestAssertError(t *testing.T) {
	t.Parallel()

	AssertError(t, errors.New("fail"), "should have error")
}

func TestAssertEqual(t *testing.T) {
	t.Parallel()

	AssertEqual(t, 42, 42, "number")
	AssertEqual(t, "hello", "hello", "string")
}

func TestAssertContains(t *testing.T) {
	t.Parallel()

	AssertContains(t, "hello world", "world", "should contain")
}

func TestAssertNotContains(t *testing.T) {
	t.Parallel()

	AssertNotContains(t, "hello world", "xyz", "should not contain")
}

func TestFakeStore_AppendBatchFn(t *testing.T) {
	t.Parallel()

	called := false
	store := NewFakeStore().AppendBatchFn(func(_ event.AggregateType, _ id.AggregateID, _ []event.Event) error {
		called = true
		return nil
	})

	err := store.AppendBatch(context.Background(), "User", id.NewAggregateID(), nil)
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}
	if !called {
		t.Fatal("AppendBatchFn not called")
	}
}

func TestFakeStore_LoadToVersionFn(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	called := false
	store := NewFakeStore().LoadToVersionFn(func(_ event.AggregateType, _ id.AggregateID, _ event.Version) ([]event.Event, error) {
		called = true
		return nil, nil
	})

	_, err := store.LoadToVersion(context.Background(), "User", aggID, 3)
	if err != nil {
		t.Fatalf("LoadToVersion: %v", err)
	}
	if !called {
		t.Fatal("LoadToVersionFn not called")
	}
}

func TestFakeStore_LoadToTimestampFn(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	called := false
	store := NewFakeStore().LoadToTimestampFn(func(_ event.AggregateType, _ id.AggregateID, _ time.Time) ([]event.Event, error) {
		called = true
		return nil, nil
	})

	_, err := store.LoadToTimestamp(context.Background(), "User", aggID, time.Now())
	if err != nil {
		t.Fatalf("LoadToTimestamp: %v", err)
	}
	if !called {
		t.Fatal("LoadToTimestampFn not called")
	}
}
