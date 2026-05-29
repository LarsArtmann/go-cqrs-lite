package middleware

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func testTracerWithRecorder() (trace.Tracer, *tracetest.SpanRecorder) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	return provider.Tracer(cqrsotel.ComponentTracer("middleware")), recorder
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
	if attrs[cqrsotel.AttrMessageKind] != "command" {
		t.Errorf("expected message.kind 'command', got %v", attrs[cqrsotel.AttrMessageKind])
	}

	if attrs[cqrsotel.AttrCommandType] != "test.cmd" {
		t.Errorf("expected command.type 'test.cmd', got %v", attrs[cqrsotel.AttrCommandType])
	}

	if _, ok := attrs[cqrsotel.AttrAggregateID]; !ok {
		t.Error("expected aggregate.id attribute to be set")
	}

	assertSpanStatusUnset(t, span)
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

	evt, err := testhelpers.NewTestEvent()
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
	if attrs[cqrsotel.AttrMessageKind] != "event" {
		t.Errorf("expected message.kind 'event', got %v", attrs[cqrsotel.AttrMessageKind])
	}

	if attrs[cqrsotel.AttrEventType] != "test.evt" {
		t.Errorf("expected event.type 'test.evt', got %v", attrs[cqrsotel.AttrEventType])
	}

	if _, ok := attrs[cqrsotel.AttrAggregateID]; !ok {
		t.Error("expected aggregate.id attribute to be set")
	}

	if _, ok := attrs[cqrsotel.AttrAggregateType]; !ok {
		t.Error("expected aggregate.type attribute to be set")
	}

	if _, ok := attrs[cqrsotel.AttrAggregateVersion]; !ok {
		t.Error("expected aggregate.version attribute to be set")
	}
}

func TestEventTracing_Error(t *testing.T) {
	t.Parallel()

	tracer, recorder := testTracerWithRecorder()
	mw := EventTracing(tracer)
	handler := mw(testhelpers.FailingEventHandler("boom"))

	evt, err := testhelpers.NewTestEvent()
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
	handler := mw(testhelpers.NoopQueryHandler())

	result, err := handler(context.Background(), &testQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}

	spans := recorder.Ended()
	testhelpers.AssertLenFatal(t, "spans", spans, 1)

	span := spans[0]
	if span.Name() != "query.handle" {
		t.Errorf("expected span name 'query.handle', got %s", span.Name())
	}

	attrs := attributeMap(span.Attributes())
	if attrs[cqrsotel.AttrMessageKind] != "query" {
		t.Errorf("expected message.kind 'query', got %v", attrs[cqrsotel.AttrMessageKind])
	}

	if attrs[cqrsotel.AttrQueryType] != "test.query" {
		t.Errorf("expected query.type 'test.query', got %v", attrs[cqrsotel.AttrQueryType])
	}
}

func TestQueryTracing_Error(t *testing.T) {
	t.Parallel()

	tracer, recorder := testTracerWithRecorder()
	mw := QueryTracing(tracer)
	handler := mw(testhelpers.FailingQueryHandler("boom"))

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

func TestEventPublishTracing_Success(t *testing.T) {
	t.Parallel()

	tracer, recorder := testTracerWithRecorder()
	mw := EventPublishTracing(tracer)
	publisher := mw(testhelpers.NoopEventPublisher())

	evt1, err := testhelpers.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	evt2, err := testhelpers.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = publisher.Publish(context.Background(), evt1, evt2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := recorder.Ended()
	testhelpers.AssertLenFatal(t, "spans", spans, 1)

	span := spans[0]
	if span.Name() != "event.publish" {
		t.Errorf("expected span name 'event.publish', got %s", span.Name())
	}

	if span.SpanKind() != trace.SpanKindProducer {
		t.Errorf("expected Producer span kind, got %v", span.SpanKind())
	}

	attrs := attributeMap(span.Attributes())
	if attrs[cqrsotel.AttrMessageKind] != "event" {
		t.Errorf("expected message.kind 'event', got %v", attrs[cqrsotel.AttrMessageKind])
	}

	if attrs[cqrsotel.AttrEventCount] != int64(2) {
		t.Errorf("expected event.count 2, got %v", attrs[cqrsotel.AttrEventCount])
	}

	if span.Status().Code != codes.Unset {
		t.Errorf("expected unset status on success, got %v", span.Status().Code)
	}
}

func TestEventPublishTracing_Error(t *testing.T) {
	t.Parallel()

	tracer, recorder := testTracerWithRecorder()
	mw := EventPublishTracing(tracer)
	publisher := mw(testhelpers.FailingEventPublisher("publish failed"))

	evt, err := testhelpers.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = publisher.Publish(context.Background(), evt)
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
