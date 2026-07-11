package middleware

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestCommandMetrics_Success(t *testing.T) {
	t.Parallel()

	metrics := &eventtest.FakeMetrics{}
	mw := CommandMetrics(metrics)

	handler := mw(NoopCommandHandler())

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	eventtest.AssertLenFatal(t, "metric records", metrics.Records, 1)

	if metrics.Records[0] != "command_success" {
		t.Errorf("expected command_success, got %s", metrics.Records[0])
	}
}

func TestCommandMetrics_Error(t *testing.T) {
	t.Parallel()

	metrics := &eventtest.FakeMetrics{}
	mw := CommandMetrics(metrics)

	handler := mw(failingCommandHandler("middleware failure"))

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error")
	}

	eventtest.AssertLenFatal(t, "metric records", metrics.Records, 1)

	if metrics.Records[0] != "command_error" {
		t.Errorf("expected command_error, got %s", metrics.Records[0])
	}
}

func TestEventMetrics_Success(t *testing.T) {
	t.Parallel()

	metrics := &eventtest.FakeMetrics{}
	mw := EventMetrics(metrics)

	handler := mw(eventtest.NoopEventHandler())

	evt, err := eventtest.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	eventtest.AssertLenFatal(t, "metric records", metrics.Records, 1)

	if metrics.Records[0] != "event_success" {
		t.Errorf("expected event_success, got %s", metrics.Records[0])
	}
}

func TestEventMetrics_Error(t *testing.T) {
	t.Parallel()

	metrics := &eventtest.FakeMetrics{}
	mw := EventMetrics(metrics)

	handler := mw(eventtest.FailingEventHandler("middleware failure"))

	evt, err := eventtest.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error")
	}

	eventtest.AssertLenFatal(t, "metric records", metrics.Records, 1)

	if metrics.Records[0] != "event_error" {
		t.Errorf("expected event_error, got %s", metrics.Records[0])
	}
}

func TestQueryMetrics_Success(t *testing.T) {
	t.Parallel()

	metrics := &eventtest.FakeMetrics{}
	mw := QueryMetrics(metrics)

	handler := mw(noopQueryHandler())

	_, err := handler(context.Background(), &testQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	eventtest.AssertLenFatal(t, "metric records", metrics.Records, 1)

	if metrics.Records[0] != "query_success" {
		t.Errorf("expected query_success, got %s", metrics.Records[0])
	}
}

func TestQueryMetrics_Error(t *testing.T) {
	t.Parallel()

	metrics := &eventtest.FakeMetrics{}
	mw := QueryMetrics(metrics)

	handler := mw(failingQueryHandler("fail"))

	_, err := handler(context.Background(), &testQuery{})
	if err == nil {
		t.Fatal("expected error")
	}

	eventtest.AssertLenFatal(t, "metric records", metrics.Records, 1)

	if metrics.Records[0] != "query_error" {
		t.Errorf("expected query_error, got %s", metrics.Records[0])
	}
}
