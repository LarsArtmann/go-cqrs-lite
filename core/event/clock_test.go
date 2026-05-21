package event_test

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
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
		t.Errorf("OccurredAt = %v, want %v (WithOccurredAt should override)", evt.OccurredAt(), overrideTime)
	}
}

func TestWithClock_BatchNewEvents(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 20, 14, 0, 0, 0, time.UTC)
	clock := func() time.Time { return fixedTime }
	aggID := id.NewAggregateID()

	events, err := event.NewEvents(
		aggID,
		"Order",
		0,
		[]event.Type{"order.created", "order.confirmed"},
		[]any{
			map[string]string{"item": "widget"},
			map[string]bool{"confirmed": true},
		},
		event.WithClock(clock),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, evt := range events {
		if !evt.OccurredAt().Equal(fixedTime) {
			t.Errorf("event[%d] OccurredAt = %v, want %v", i, evt.OccurredAt(), fixedTime)
		}
	}
}

func TestWithClock_BuilderPattern(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return fixedTime }

	evt, err := event.NewBuilder(
		"OrderPlaced",
		id.NewAggregateID(),
		"Order",
		1,
	).WithOptions(
		event.WithClock(clock),
	).Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !evt.OccurredAt().Equal(fixedTime) {
		t.Errorf("OccurredAt = %v, want %v", evt.OccurredAt(), fixedTime)
	}
}

func TestDefaultClock_IsTimeNow(t *testing.T) {
	t.Parallel()

	if event.DefaultClock == nil {
		t.Fatal("DefaultClock should not be nil")
	}

	result := event.DefaultClock()
	if result.IsZero() {
		t.Error("DefaultClock() should return non-zero time")
	}
}
