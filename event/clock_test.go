package event_test

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
)

func TestWithClock_DeterministicTime(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	clock := func() time.Time { return fixedTime }

	evt, err := event.NewEvent(
		"UserCreated",
		id.NewAggregateID(),
		"User",
		1,
		nil,
		event.WithClock(clock),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !evt.OccurredAt().Equal(fixedTime) {
		t.Errorf("OccurredAt = %v, want %v", evt.OccurredAt(), fixedTime)
	}
}

func TestWithClock_DefaultUsesTimeNow(t *testing.T) {
	t.Parallel()

	before := time.Now()

	evt, err := event.NewEvent(
		"UserCreated",
		id.NewAggregateID(),
		"User",
		1,
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	after := time.Now()

	if evt.OccurredAt().Before(before) || evt.OccurredAt().After(after) {
		t.Errorf("OccurredAt = %v, expected between %v and %v", evt.OccurredAt(), before, after)
	}
}

func TestWithClock_WithOccurredAtTakesPrecedence(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	overrideTime := time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC)
	clock := func() time.Time { return fixedTime }

	evt, err := event.NewEvent(
		"UserCreated",
		id.NewAggregateID(),
		"User",
		1,
		nil,
		event.WithClock(clock),
		event.WithOccurredAt(overrideTime),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !evt.OccurredAt().Equal(overrideTime) {
		t.Errorf(
			"OccurredAt = %v, want %v (WithOccurredAt should override)",
			evt.OccurredAt(),
			overrideTime,
		)
	}
}

func TestWithClock_BatchNewEvents(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 20, 14, 0, 0, 0, time.UTC)
	clock := func() time.Time { return fixedTime }
	aggID := id.NewAggregateID()

	evt1, err := event.NewEvent("order.created", aggID, "Order", 1,
		[]byte(`{"item":"widget"}`), event.WithClock(clock))
	if err != nil {
		t.Fatalf("NewEvent 1: %v", err)
	}

	evt2, err := event.NewEvent("order.confirmed", aggID, "Order", 2,
		[]byte(`{"confirmed":true}`), event.WithClock(clock))
	if err != nil {
		t.Fatalf("NewEvent 2: %v", err)
	}

	for i, evt := range []event.Event{evt1, evt2} {
		if !evt.OccurredAt().Equal(fixedTime) {
			t.Errorf("event[%d] OccurredAt = %v, want %v", i, evt.OccurredAt(), fixedTime)
		}
	}
}

func TestDefaultClock_IsTimeNow(t *testing.T) {
	t.Parallel()

	before := time.Now().UTC()
	evt, err := event.NewEvent("TestEvent", id.NewAggregateID(), "Test", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	after := time.Now().UTC()
	occurred := evt.OccurredAt().UTC()

	if occurred.Before(before) || occurred.After(after) {
		t.Errorf("OccurredAt = %v, expected between %v and %v", occurred, before, after)
	}
}
