package integration_test

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/middleware/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
)

func TestOTel_CommandDispatch_EmitsSpans(t *testing.T) {
	t.Parallel()

	recorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	defer func() { _ = tracerProvider.Shutdown(context.Background()) }()

	tracer := tracerProvider.Tracer(cqrsotel.ComponentTracer("test"))

	cmdDispatcher := command.NewDispatcher()
	cmdDispatcher.Use(middleware.CommandTracing(tracer))

	cmdType := command.Type("CreateOrder")
	aggID := id.NewAggregateID()

	err := cmdDispatcher.Register(cmdType, func(_ context.Context, _ command.Command) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	cmd, err := command.New(cmdType, aggID)
	if err != nil {
		t.Fatalf("New command: %v", err)
	}

	err = cmdDispatcher.Dispatch(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	spans := recorder.Ended()
	if len(spans) == 0 {
		t.Fatal("expected at least one span from command dispatch")
	}

	found := false
	for _, s := range spans {
		if s.Name() == "command.handle" {
			found = true

			break
		}
	}

	if !found {
		names := make([]string, len(spans))
		for i, s := range spans {
			names[i] = s.Name()
		}

		t.Errorf("expected 'command.dispatch' span, got: %v", names)
	}
}

func TestOTel_EventBus_EmitsSpans(t *testing.T) {
	t.Parallel()

	recorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	defer func() { _ = tracerProvider.Shutdown(context.Background()) }()

	tracer := tracerProvider.Tracer(cqrsotel.ComponentTracer("test"))

	bus := eventtest.NewFakeBus()
	defer bus.Close()

	_ = bus.Use(middleware.EventTracing(tracer))

	received := false
	_ = bus.Subscribe("OrderCreated", func(_ context.Context, _ event.Event) error {
		received = true

		return nil
	})

	aggID := id.NewAggregateID()
	evt, err := event.NewEvent("OrderCreated", aggID, "Order", 1, []byte("{}"))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	err = bus.Publish(context.Background(), evt)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if !received {
		t.Fatal("expected event to be received")
	}

	spans := recorder.Ended()
	if len(spans) == 0 {
		t.Fatal("expected at least one span from event bus")
	}
}
