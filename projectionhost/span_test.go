package projectionhost_test

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// TestHost_OTelSpans_VerifyNamesAndAttributes verifies that the managed host
// emits correctly-named spans with the expected attributes:
//   - projectionhost.drain (Internal span, carries cqrs.projection.name)
//   - projectionhost.handle_event (Consumer span, carries projection name,
//     event type, and event ID)
func TestHost_OTelSpans_VerifyNamesAndAttributes(t *testing.T) {
	// NOT parallel — mutates global TracerProvider.
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer tp.Shutdown(context.Background())

	origTP := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(origTP)

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	proj := &countingProjection{
		name:       "span-test",
		eventTypes: []event.Type{"test.created"},
	}

	host, err := projectionhost.New(
		journal, cpStore,
		projectionhost.WithBatchSize(10),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := host.Register(proj); err != nil {
		t.Fatalf("Register: %v", err)
	}

	evt := makeEvent("test.created")
	journal.append(evt)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_ = host.Start(ctx)
	}()

	requireEventually(t, 3*time.Second, func() bool {
		return proj.count.Load() >= 1
	})

	cancel()
	_ = host.Stop()

	tp.ForceFlush(context.Background())
	spans := exporter.GetSpans()

	var drainSpan, handleSpan *tracetest.SpanStub

	for i := range spans {
		switch spans[i].Name {
		case "projectionhost.drain":
			drainSpan = &spans[i]
		case "projectionhost.handle_event":
			handleSpan = &spans[i]
		}
	}

	if drainSpan == nil {
		names := make([]string, 0, len(spans))
		for _, s := range spans {
			names = append(names, s.Name)
		}
		t.Fatalf("projectionhost.drain span not found; got spans: %v", names)
	}

	if handleSpan == nil {
		names := make([]string, 0, len(spans))
		for _, s := range spans {
			names = append(names, s.Name)
		}
		t.Fatalf("projectionhost.handle_event span not found; got spans: %v", names)
	}

	// Verify drain span attributes.
	drainAttrs := spanAttrMap(drainSpan.Attributes)
	if drainAttrs["cqrs.projection.name"] != "span-test" {
		t.Errorf("drain span: expected projection name 'span-test', got %q",
			drainAttrs["cqrs.projection.name"])
	}

	// Verify handle_event span attributes.
	handleAttrs := spanAttrMap(handleSpan.Attributes)
	if handleAttrs["cqrs.projection.name"] != "span-test" {
		t.Errorf("handle_event span: expected projection name 'span-test', got %q",
			handleAttrs["cqrs.projection.name"])
	}
	if handleAttrs["cqrs.event.type"] != "test.created" {
		t.Errorf("handle_event span: expected event type 'test.created', got %q",
			handleAttrs["cqrs.event.type"])
	}
	if handleAttrs["cqrs.event.id"] == "" {
		t.Error("handle_event span: expected non-empty event ID")
	}
}

func spanAttrMap(attrs []attribute.KeyValue) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, kv := range attrs {
		m[string(kv.Key)] = kv.Value.AsString()
	}

	return m
}
