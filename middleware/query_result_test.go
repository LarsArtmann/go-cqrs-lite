package middleware

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

func TestQueryResultPropagation_Recovery(t *testing.T) {
	t.Parallel()

	mw := QueryRecovery()
	handler := mw(func(_ context.Context, _ query.Query) (any, error) {
		return "hello", nil
	})

	result, err := handler(context.Background(), &testQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "hello" {
		t.Errorf("expected 'hello', got %v (result was not propagated!)", result)
	}
}

func TestQueryResultPropagation_Logging(t *testing.T) {
	t.Parallel()

	logger, _ := newTestLogger()
	mw := QueryLogging(logger)
	handler := mw(func(_ context.Context, _ query.Query) (any, error) {
		return 42, nil
	})

	result, err := handler(context.Background(), &testQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 42 {
		t.Errorf("expected 42, got %v (result was not propagated!)", result)
	}
}

func TestQueryResultPropagation_Metrics(t *testing.T) {
	t.Parallel()

	metrics := &eventtest.FakeMetrics{}
	mw := QueryMetrics(metrics)
	handler := mw(func(_ context.Context, _ query.Query) (any, error) {
		return struct{ Name string }{"test"}, nil
	})

	result, err := handler(context.Background(), &testQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Error("expected non-nil result (result was not propagated!)")
	}
}

func TestQueryResultPropagation_Tracing(t *testing.T) {
	t.Parallel()

	tracer, _ := testTracerWithRecorder()
	mw := QueryTracing(tracer)
	handler := mw(func(_ context.Context, _ query.Query) (any, error) {
		return "traced", nil
	})

	result, err := handler(context.Background(), &testQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "traced" {
		t.Errorf("expected 'traced', got %v (result was not propagated!)", result)
	}
}

func TestQueryResultPropagation_Stacked(t *testing.T) {
	t.Parallel()

	logger, _ := newTestLogger()
	tracer, _ := testTracerWithRecorder()

	handler := func(_ context.Context, _ query.Query) (any, error) {
		return "stacked", nil
	}

	wrapped := QueryTracing(tracer)(QueryLogging(logger)(handler))

	result, err := wrapped(context.Background(), &testQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "stacked" {
		t.Errorf("expected 'stacked', got %v (result was not propagated through stack!)", result)
	}
}

func TestQueryResultPropagation_Retry(t *testing.T) {
	t.Parallel()

	config := DefaultRetryConfig()
	mw := QueryRetry(config)
	handler := mw(func(_ context.Context, _ query.Query) (any, error) {
		return "retried", nil
	})

	result, err := handler(context.Background(), &testQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "retried" {
		t.Errorf("expected 'retried', got %v (result was not propagated!)", result)
	}
}
