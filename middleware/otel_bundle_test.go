package middleware

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestNewOTelBundle(t *testing.T) {
	t.Parallel()

	bundle := newTestBundle(t)

	if bundle == nil {
		t.Fatal("expected non-nil bundle")
	}

	if bundle.Tracer() == nil {
		t.Error("expected non-nil tracer")
	}

	if bundle.Recorder() == nil {
		t.Error("expected non-nil recorder")
	}
}

func TestOTelBundle_CommandMiddleware(t *testing.T) {
	t.Parallel()

	bundle := newTestBundle(t)

	mws := bundle.Command()
	if len(mws) != 2 {
		t.Fatalf("expected 2 command middleware, got %d", len(mws))
	}

	handler := mws[0](mws[1](NoopCommandHandler()))
	cmd := &testCommand{streamID: id.NewStreamID()}

	err := handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOTelBundle_EventMiddleware(t *testing.T) {
	t.Parallel()

	bundle := newTestBundle(t)

	mws := bundle.Event()
	if len(mws) != 2 {
		t.Fatalf("expected 2 event middleware, got %d", len(mws))
	}

	handler := mws[0](mws[1](eventtest.NoopEventHandler()))

	evt, err := eventtest.NewTestEvent()
	if err != nil {
		t.Fatalf("create test event: %v", err)
	}

	err = handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOTelBundle_QueryMiddleware(t *testing.T) {
	t.Parallel()

	bundle := newTestBundle(t)

	mws := bundle.Query()
	if len(mws) != 2 {
		t.Fatalf("expected 2 query middleware, got %d", len(mws))
	}

	handler := mws[0](mws[1](noopQueryHandler()))
	q := &testQuery{}

	_, err := handler(context.Background(), q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOTelBundle_PublishMiddleware(t *testing.T) {
	t.Parallel()

	bundle := newTestBundle(t)

	mws := bundle.Publish()
	if len(mws) != 1 {
		t.Fatalf("expected 1 publish middleware, got %d", len(mws))
	}
}

func TestOTelBundle_ProducesSpans(t *testing.T) {
	t.Parallel()

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	defer tp.Shutdown(context.Background())

	meter := metric.NewMeterProvider().Meter("test")

	bundle, err := NewOTelBundle(tp.Tracer("test"), meter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mws := bundle.Command()
	handler := mws[0](mws[1](NoopCommandHandler()))
	cmd := &testCommand{streamID: id.NewStreamID()}

	_ = handler(context.Background(), cmd)

	tp.ForceFlush(context.Background())

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	if spans[0].Name != "command.handle" {
		t.Errorf("expected span name 'command.handle', got %q", spans[0].Name)
	}
}

func TestOTelBundle_WithMetricsDisabled(t *testing.T) {
	t.Parallel()

	tracer := sdktrace.NewTracerProvider().Tracer("test")

	bundle, err := NewOTelBundle(tracer, nil, WithMetricsDisabled())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if bundle.Recorder() != nil {
		t.Error("expected nil recorder when metrics disabled")
	}

	if len(bundle.Command()) != 1 {
		t.Errorf("expected 1 command middleware (tracing only), got %d", len(bundle.Command()))
	}

	if len(bundle.Event()) != 1 {
		t.Errorf("expected 1 event middleware (tracing only), got %d", len(bundle.Event()))
	}

	if len(bundle.Query()) != 1 {
		t.Errorf("expected 1 query middleware (tracing only), got %d", len(bundle.Query()))
	}
}

func TestOTelBundle_NilMeterWithoutDisableReturnsError(t *testing.T) {
	t.Parallel()

	tracer := sdktrace.NewTracerProvider().Tracer("test")

	_, err := NewOTelBundle(tracer, nil)
	if err == nil {
		t.Fatal("expected error when meter is nil and metrics not disabled")
	}
}

func TestOTelBundle_CorrelationEnricher(t *testing.T) {
	t.Parallel()

	bundle := newTestBundle(t)

	enricher := bundle.CorrelationEnricher()
	if enricher == nil {
		t.Fatal("expected non-nil enricher")
	}

	// Without baggage correlation ID, returns nil options.
	opts := enricher(context.Background())
	if opts != nil {
		t.Errorf("expected nil options without baggage, got %v", opts)
	}
}
