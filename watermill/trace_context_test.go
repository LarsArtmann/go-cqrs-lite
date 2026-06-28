package watermill_test

import (
	"context"
	"testing"
	"time"

	gochannel "github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
	wm "github.com/larsartmann/go-cqrs-lite/watermill/v3"
)

// TestTraceContext_PropagationLinksSpans verifies that W3C trace context
// injected on publish is extracted on consume, linking producer and consumer
// spans into a single trace tree across the Watermill message boundary.
func TestTraceContext_PropagationLinksSpans(t *testing.T) {
	// NOT parallel — mutates global propagator and tracer provider.
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer tp.Shutdown(context.Background())

	origProp := otel.GetTextMapPropagator()
	origTP := otel.GetTracerProvider()
	defer func() {
		otel.SetTextMapPropagator(origProp)
		otel.SetTracerProvider(origTP)
	}()

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	pubSub := gochannel.NewGoChannel(gochannel.Config{}, nil)
	defer pubSub.Close()

	msgs, err := pubSub.Subscribe(context.Background(), "trace.test")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	eventPub := wm.NewEventPublisher(pubSub, "trace.test")

	// Create a parent span, then publish under it.
	ctx, parentSpan := tp.Tracer("test").Start(context.Background(), "parent.operation")

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent("trace.test.event", aggID, "Test", 1, []byte(`{}`))

	if err := eventPub.Publish(ctx, evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	parentSpan.End()

	// Consume the message.
	select {
	case msg := <-msgs:
		if msg == nil {
			t.Fatal("received nil message")
		}

		// Extract trace context from message metadata.
		consumeCtx := wm.ExtractContext(msg.Context(), msg)

		// Create a consumer span under the extracted context.
		tracer := cqrsotel.NewTracer("test")
		consumeCtx, consumeSpan := cqrsotel.StartSpan(
			consumeCtx, tracer, "consumer.handle",
			cqrsotel.SpanKindConsumer,
		)
		consumeSpan.End()

		msg.Ack()

		tp.ForceFlush(context.Background())

		spans := exporter.GetSpans()
		if len(spans) < 2 {
			t.Fatalf("expected at least 2 spans, got %d", len(spans))
		}

		// Find the consumer span and verify it shares the parent's trace.
		var consumerSpan *tracetest.SpanStub
		for i := range spans {
			if spans[i].Name == "consumer.handle" {
				consumerSpan = &spans[i]
			}
		}

		if consumerSpan == nil {
			t.Fatal("consumer.handle span not found")
		}

		if consumerSpan.SpanContext.TraceID() != parentSpan.SpanContext().TraceID() {
			t.Errorf("consumer span trace ID %s does not match parent trace ID %s",
				consumerSpan.SpanContext.TraceID(), parentSpan.SpanContext().TraceID())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}

// TestTraceContext_MessageCarriesTraceparent verifies that publishing injects
// a traceparent header into the message metadata.
func TestTraceContext_MessageCarriesTraceparent(t *testing.T) {
	// NOT parallel — mutates global propagator.
	origProp := otel.GetTextMapPropagator()
	defer otel.SetTextMapPropagator(origProp)

	otel.SetTextMapPropagator(propagation.TraceContext{})

	pubSub := gochannel.NewGoChannel(gochannel.Config{}, nil)
	defer pubSub.Close()

	msgs, err := pubSub.Subscribe(context.Background(), "traceparent.test")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(sdktrace.NewTracerProvider())

	eventPub := wm.NewEventPublisher(pubSub, "traceparent.test")

	ctx, span := tp.Tracer("test").Start(context.Background(), "publish.parent")

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent("tp.test", aggID, "Test", 1, []byte(`{}`))

	if err := eventPub.Publish(ctx, evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	span.End()

	select {
	case msg := <-msgs:
		tp, ok := msg.Metadata["traceparent"]
		if !ok {
			t.Fatal("expected traceparent in message metadata")
		}

		if tp == "" {
			t.Error("traceparent metadata is empty")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}

var consumeTimeout = consumeTimeoutDuration()

func consumeTimeoutDuration() (ch <-chan struct{}) {
	c := make(chan struct{})
	close(c)

	return c
}
