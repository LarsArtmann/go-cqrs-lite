package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func testTracerWithRecorder() (trace.Tracer, *tracetest.SpanRecorder) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	return provider.Tracer(instrumentationName), recorder
}

func TestCommandTracing_Success(t *testing.T) {
	t.Parallel()

	tracer, recorder := testTracerWithRecorder()
	mw := CommandTracing(tracer)
	handler := mw(testhelpers.NoopCommandHandler())

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := recorder.Ended()
	testhelpers.AssertLenFatal(t, "spans", spans, 1)

	span := spans[0]
	if span.Name() != "command.handle" {
		t.Errorf("expected span name 'command.handle', got %s", span.Name())
	}

	attrs := attributeMap(span.Attributes())
	if attrs["cqrs.message.kind"] != "command" {
		t.Errorf("expected message.kind 'command', got %v", attrs["cqrs.message.kind"])
	}

	if span.Status().Code != codes.Unset {
		t.Errorf("expected unset status on success, got %v", span.Status().Code)
	}
}

func TestCommandTracing_Error(t *testing.T) {
	t.Parallel()

	tracer, recorder := testTracerWithRecorder()
	mw := CommandTracing(tracer)
	handler := mw(testhelpers.FailingCommandHandler("boom"))

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error")
	}

	spans := recorder.Ended()
	testhelpers.AssertLenFatal(t, "spans", spans, 1)

	span := spans[0]
	if span.Status().Code != codes.Error {
		t.Errorf("expected error status, got %v", span.Status().Code)
	}
}

func TestEventTracing_Success(t *testing.T) {
	t.Parallel()

	tracer, recorder := testTracerWithRecorder()
	mw := EventTracing(tracer)
	handler := mw(testhelpers.NoopEventHandler())

	evt, err := event.NewEvent("test.evt", id.NewAggregateID(), "Test", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := recorder.Ended()
	testhelpers.AssertLenFatal(t, "spans", spans, 1)

	span := spans[0]
	if span.Name() != "event.handle" {
		t.Errorf("expected span name 'event.handle', got %s", span.Name())
	}

	attrs := attributeMap(span.Attributes())
	if attrs["cqrs.message.kind"] != "event" {
		t.Errorf("expected message.kind 'event', got %v", attrs["cqrs.message.kind"])
	}
}

func TestEventTracing_Error(t *testing.T) {
	t.Parallel()

	tracer, recorder := testTracerWithRecorder()
	mw := EventTracing(tracer)
	handler := mw(testhelpers.FailingEventHandler("boom"))

	evt, err := event.NewEvent("test.evt", id.NewAggregateID(), "Test", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error")
	}

	spans := recorder.Ended()
	testhelpers.AssertLenFatal(t, "spans", spans, 1)

	span := spans[0]
	if span.Status().Code != codes.Error {
		t.Errorf("expected error status, got %v", span.Status().Code)
	}
}

func TestQueryTracing_Success(t *testing.T) {
	t.Parallel()

	tracer, recorder := testTracerWithRecorder()
	mw := QueryTracing(tracer)
	handler := mw(func(_ context.Context, _ query.Query) (any, error) {
		return "result", nil
	})

	result, err := handler(context.Background(), &testQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "result" {
		t.Errorf("expected 'result', got %v", result)
	}

	spans := recorder.Ended()
	testhelpers.AssertLenFatal(t, "spans", spans, 1)

	span := spans[0]
	if span.Name() != "query.handle" {
		t.Errorf("expected span name 'query.handle', got %s", span.Name())
	}

	attrs := attributeMap(span.Attributes())
	if attrs["cqrs.message.kind"] != "query" {
		t.Errorf("expected message.kind 'query', got %v", attrs["cqrs.message.kind"])
	}
}

func TestQueryTracing_Error(t *testing.T) {
	t.Parallel()

	tracer, recorder := testTracerWithRecorder()
	mw := QueryTracing(tracer)
	handler := mw(func(_ context.Context, _ query.Query) (any, error) {
		return nil, errors.New("boom")
	})

	_, err := handler(context.Background(), &testQuery{})
	if err == nil {
		t.Fatal("expected error")
	}

	spans := recorder.Ended()
	testhelpers.AssertLenFatal(t, "spans", spans, 1)

	span := spans[0]
	if span.Status().Code != codes.Error {
		t.Errorf("expected error status, got %v", span.Status().Code)
	}
}

func attributeMap(attrs []attribute.KeyValue) map[string]any {
	m := make(map[string]any, len(attrs))
	for _, attr := range attrs {
		m[string(attr.Key)] = attr.Value.AsInterface()
	}

	return m
}
