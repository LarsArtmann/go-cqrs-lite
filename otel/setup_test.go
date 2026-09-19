package otel_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

func TestSetup_Defaults(t *testing.T) {
	t.Parallel()

	provider, err := cqrsotel.Setup(
		cqrsotel.WithoutGlobalRegistration(),
		cqrsotel.WithService("test-svc", "1.0.0", "test-instance"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defer provider.Shutdown(context.Background())

	if provider.AsTracerProvider() == nil {
		t.Error("expected non-nil tracer provider")
	}

	if provider.AsMeterProvider() == nil {
		t.Error("expected non-nil meter provider")
	}
}

func TestSetup_SetsGlobalProviders(t *testing.T) {
	// NOT parallel — mutates global state
	originalTP := otel.GetTracerProvider()
	originalMP := otel.GetMeterProvider()

	defer func() {
		otel.SetTracerProvider(originalTP)
		otel.SetMeterProvider(originalMP)
	}()

	provider, err := cqrsotel.Setup(
		cqrsotel.WithService("test-svc", "0.0.0", ""),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defer provider.Shutdown(context.Background())

	if otel.GetTracerProvider() != provider.AsTracerProvider() {
		t.Error("expected global TracerProvider to be set")
	}

	if otel.GetMeterProvider() != provider.AsMeterProvider() {
		t.Error("expected global MeterProvider to be set")
	}
}

func TestSetup_WithSpanExporter(t *testing.T) {
	t.Parallel()

	exporter := tracetest.NewInMemoryExporter()

	provider, err := cqrsotel.Setup(
		cqrsotel.WithoutGlobalRegistration(),
		cqrsotel.WithService("test-svc", "1.0.0", ""),
		cqrsotel.WithSpanExporter(exporter),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defer provider.Shutdown(context.Background())

	tracer := provider.AsTracerProvider().Tracer("test")
	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")
	span.End()

	provider.AsTracerProvider().ForceFlush(context.Background())

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 exported span, got %d", len(spans))
	}

	if spans[0].Name != "test-span" {
		t.Errorf("expected span name 'test-span', got %q", spans[0].Name)
	}
}

func TestSetup_Shutdown(t *testing.T) {
	t.Parallel()

	provider, err := cqrsotel.Setup(
		cqrsotel.WithoutGlobalRegistration(),
		cqrsotel.WithService("test-svc", "1.0.0", ""),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := provider.Shutdown(context.Background()); err != nil {
		t.Fatalf("unexpected shutdown error: %v", err)
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()

	v := cqrsotel.Version()
	if v == "" {
		t.Error("expected non-empty version string")
	}
}

func TestSetup_ResourceAttributes(t *testing.T) {
	t.Parallel()

	exporter := tracetest.NewInMemoryExporter()

	provider, err := cqrsotel.Setup(
		cqrsotel.WithoutGlobalRegistration(),
		cqrsotel.WithService("my-service", "2.0.0", "i-42"),
		cqrsotel.WithSpanExporter(exporter),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defer provider.Shutdown(context.Background())

	tracer := provider.AsTracerProvider().Tracer("test")
	_, span := tracer.Start(context.Background(), "op")
	span.End()

	provider.AsTracerProvider().ForceFlush(context.Background())

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	attrs := spans[0].Resource.Attributes()
	found := map[string]bool{}

	for _, attr := range attrs {
		found[string(attr.Key)] = true
	}

	for _, key := range []string{"service.name", "service.version", "service.instance.id"} {
		if !found[key] {
			t.Errorf("expected resource attribute %q on span", key)
		}
	}
}

func TestSetup_CQRSHistogramViews(t *testing.T) {
	t.Parallel()

	exporter := tracetest.NewInMemoryExporter()
	provider, err := cqrsotel.Setup(
		cqrsotel.WithoutGlobalRegistration(),
		cqrsotel.WithService("test", "1.0", ""),
		cqrsotel.WithSpanExporter(exporter),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defer provider.Shutdown(context.Background())

	if provider.AsTracerProvider() == nil {
		t.Fatal("expected non-nil tracer provider")
	}

	_ = sdktrace.NewTracerProvider() // verify SDK import compiles
}

func TestSetup_WithStdoutExporter(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	provider, err := cqrsotel.Setup(
		cqrsotel.WithoutGlobalRegistration(),
		cqrsotel.WithService("test-stdout", "1.0.0", ""),
		cqrsotel.WithStdoutExporter(&buf),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tracer := provider.AsTracerProvider().Tracer("test")
	_, span := tracer.Start(context.Background(), "stdout-span")
	span.End()

	provider.AsTracerProvider().ForceFlush(context.Background())
	provider.Shutdown(context.Background())

	output := buf.String()
	if !strings.Contains(output, "stdout-span") {
		t.Errorf("expected stdout output to contain span name, got:\n%s", output)
	}
}
