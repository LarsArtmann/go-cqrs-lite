package http

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

func TestSSE_FanoutSpanCarriesEventAttrs(t *testing.T) {
	// NOT parallel — mutates global TracerProvider.
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer tp.Shutdown(context.Background())

	origTP := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(origTP)

	bus := eventtest.NewFakeBus()
	defer bus.Close()

	broker, err := NewSSEBroker(bus)
	if err != nil {
		t.Fatalf("NewSSEBroker: %v", err)
	}
	defer broker.Close()

	broker.AddClient("client-1")

	aggID := id.NewAggregateID()
	evt, err := event.NewEvent("UserCreated", aggID, "User", 1, []byte(`{}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	if err := bus.Publish(context.Background(), evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	tp.ForceFlush(context.Background())
	spans := exporter.GetSpans()

	var fanout *tracetest.SpanStub

	for i := range spans {
		if spans[i].Name == "sse.fanout" {
			fanout = &spans[i]
		}
	}

	if fanout == nil {
		names := make([]string, 0, len(spans))
		for _, s := range spans {
			names = append(names, s.Name)
		}
		t.Fatalf("sse.fanout span not found, got: %v", names)
	}

	attrs := attrMap(fanout.Attributes)
	if attrs[cqrsotel.AttrEventType] != "UserCreated" {
		t.Errorf("expected event type attr = UserCreated, got %v", attrs[cqrsotel.AttrEventType])
	}

	if attrs[cqrsotel.AttrMessageKind] != cqrsotel.KindEvent {
		t.Errorf("expected message kind = event, got %v", attrs[cqrsotel.AttrMessageKind])
	}
}

func attrMap(attrs []attribute.KeyValue) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, kv := range attrs {
		m[string(kv.Key)] = kv.Value.AsString()
	}

	return m
}
