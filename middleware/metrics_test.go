package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestCommandMetrics_Success(t *testing.T) {
	t.Parallel()

	metrics := &testhelpers.TestMetrics{}
	mw := CommandMetrics(metrics)

	handler := mw(testhelpers.NoopCommandHandler())

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	testhelpers.AssertLenFatal(t, "metric records", metrics.Records, 1)

	if metrics.Records[0] != "command_success" {
		t.Errorf("expected command_success, got %s", metrics.Records[0])
	}
}

func TestCommandMetrics_Error(t *testing.T) {
	t.Parallel()

	metrics := &testhelpers.TestMetrics{}
	mw := CommandMetrics(metrics)

	handler := mw(testhelpers.FailingCommandHandler("middleware failure"))

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error")
	}

	testhelpers.AssertLenFatal(t, "metric records", metrics.Records, 1)

	if metrics.Records[0] != "command_error" {
		t.Errorf("expected command_error, got %s", metrics.Records[0])
	}
}

func TestEventMetrics_Success(t *testing.T) {
	t.Parallel()

	metrics := &testhelpers.TestMetrics{}
	mw := EventMetrics(metrics)

	handler := mw(testhelpers.NoopEventHandler())

	evt, err := event.NewEvent("test.evt", id.NewAggregateID(), "Test", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	testhelpers.AssertLenFatal(t, "metric records", metrics.Records, 1)

	if metrics.Records[0] != "event_success" {
		t.Errorf("expected event_success, got %s", metrics.Records[0])
	}
}

func TestEventMetrics_Error(t *testing.T) {
	t.Parallel()

	metrics := &testhelpers.TestMetrics{}
	mw := EventMetrics(metrics)

	handler := mw(testhelpers.FailingEventHandler("middleware failure"))

	evt, err := event.NewEvent("test.evt", id.NewAggregateID(), "Test", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error")
	}

	testhelpers.AssertLenFatal(t, "metric records", metrics.Records, 1)

	if metrics.Records[0] != "event_error" {
		t.Errorf("expected event_error, got %s", metrics.Records[0])
	}
}

func TestQueryMetrics_Success(t *testing.T) {
	t.Parallel()

	metrics := &testhelpers.TestMetrics{}
	mw := QueryMetrics(metrics)

	handler := mw(func(_ context.Context, _ query.Query) (any, error) {
		return "result", nil
	})

	_, err := handler(context.Background(), &testQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	testhelpers.AssertLenFatal(t, "metric records", metrics.Records, 1)

	if metrics.Records[0] != "query_success" {
		t.Errorf("expected query_success, got %s", metrics.Records[0])
	}
}

func TestQueryMetrics_Error(t *testing.T) {
	t.Parallel()

	metrics := &testhelpers.TestMetrics{}
	mw := QueryMetrics(metrics)

	handler := mw(func(_ context.Context, _ query.Query) (any, error) {
		return nil, errors.New("fail")
	})

	_, err := handler(context.Background(), &testQuery{})
	if err == nil {
		t.Fatal("expected error")
	}

	testhelpers.AssertLenFatal(t, "metric records", metrics.Records, 1)

	if metrics.Records[0] != "query_error" {
		t.Errorf("expected query_error, got %s", metrics.Records[0])
	}
}
