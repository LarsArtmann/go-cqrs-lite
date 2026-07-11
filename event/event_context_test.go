package event_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
)

func TestEventContext_DeadlineMethod(t *testing.T) {
	t.Parallel()

	t.Run("no deadline returns zero time and false", func(t *testing.T) {
		t.Parallel()

		evt, err := event.NewEvent(
			"TestEvent",
			idtest.ParseAggregateID(t, "01HK154DK8FZYV2ANMQ6B0N1JY"),
			"TestAggregate",
			1,
			nil,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		deadline, ok := evt.Deadline()
		if ok {
			t.Errorf("expected ok to be false, got true")
		}
		if !deadline.IsZero() {
			t.Errorf("expected zero time, got %v", deadline)
		}
	})

	t.Run("with deadline returns time and true", func(t *testing.T) {
		t.Parallel()

		deadline := time.Now().Add(1 * time.Hour)
		evt, err := event.NewEvent(
			"TestEvent",
			idtest.ParseAggregateID(t, "01HK154DK8FZYV2ANMQ6B0N1JY"),
			"TestAggregate",
			1,
			nil,
			event.WithDeadline(deadline),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		gotDeadline, ok := evt.Deadline()
		if !ok {
			t.Errorf("expected ok to be true, got false")
		}
		if !gotDeadline.Equal(deadline) {
			t.Errorf("expected deadline %v, got %v", deadline, gotDeadline)
		}
	})
}

func TestEventContext_FromContext(t *testing.T) {
	t.Parallel()

	t.Run("context with deadline extracts deadline", func(t *testing.T) {
		t.Parallel()

		deadline := time.Now().Add(1 * time.Hour)
		ctx, cancel := context.WithDeadline(context.Background(), deadline)
		defer cancel()

		evt, err := event.NewEvent(
			"TestEvent",
			idtest.ParseAggregateID(t, "01HK154DK8FZYV2ANMQ6B0N1JY"),
			"TestAggregate",
			1,
			nil,
			event.FromContext(ctx),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		gotDeadline, ok := evt.Deadline()
		if !ok {
			t.Errorf("expected ok to be true, got false")
		}

		diff := gotDeadline.Sub(deadline)
		if diff < 0 {
			diff = -diff
		}
		if diff > 1*time.Second {
			t.Errorf("expected deadline close to %v, got %v", deadline, gotDeadline)
		}
	})

	t.Run("context without deadline is no-op", func(t *testing.T) {
		t.Parallel()

		evt, err := event.NewEvent(
			"TestEvent",
			idtest.ParseAggregateID(t, "01HK154DK8FZYV2ANMQ6B0N1JY"),
			"TestAggregate",
			1,
			nil,
			event.FromContext(context.Background()),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, ok := evt.Deadline()
		if ok {
			t.Errorf("expected no deadline, got one")
		}
	})
}

func TestEventContext_Clone(t *testing.T) {
	t.Parallel()

	t.Run("clone preserves deadline", func(t *testing.T) {
		t.Parallel()

		deadline := time.Now().Add(1 * time.Hour)
		evt, err := event.NewEvent(
			"TestEvent",
			idtest.ParseAggregateID(t, "01HK154DK8FZYV2ANMQ6B0N1JY"),
			"TestAggregate",
			1,
			nil,
			event.WithDeadline(deadline),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cloned := evt.Clone()

		gotDeadline, ok := cloned.Deadline()
		if !ok {
			t.Errorf("expected deadline on clone, got none")
		}
		if !gotDeadline.Equal(deadline) {
			t.Errorf("expected deadline %v, got %v", deadline, gotDeadline)
		}
	})
}
