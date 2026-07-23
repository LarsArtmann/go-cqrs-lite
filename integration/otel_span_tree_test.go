package integration_test

import (
	"context"
	"slices"
	"sort"
	"testing"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func setupSpanTreeTest(
	t *testing.T,
) (*tracetest.InMemoryExporter, *sdktrace.TracerProvider, *middleware.OTelBundle) {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	// Set global providers so decider/storage internal tracers (which resolve
	// via cqrsotel.NewTracer → otel.GetTracerProvider) emit spans too.
	origTP := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(origTP) })

	bundle, err := middleware.NewOTelBundle(
		tp.Tracer(cqrsotel.ComponentTracer("test")),
		sdkmetric.NewMeterProvider().Meter("test"),
	)
	if err != nil {
		t.Fatalf("new bundle: %v", err)
	}

	return exporter, tp, bundle
}

// TestOTel_SpanTree_EndToEnd verifies that a command dispatch through the
// decider produces a connected span tree: command.handle → decider.execute →
// decider.load, all sharing one trace ID with correct parent-child links.
func TestOTel_SpanTree_EndToEnd(t *testing.T) {
	// NOT parallel — mutates global TracerProvider.

	exporter, tp, bundle := setupSpanTreeTest(t)

	store := memory.NewMemoryStore()
	defer store.Close()

	bus := eventtest.NewFakeBus()
	defer bus.Close()

	bus.Use(bundle.Event()...)
	bus.UsePublish(bundle.Publish()...)

	cmdDispatcher := command.NewDispatcher()
	cmdDispatcher.Use(bundle.Command()...)

	userDecider := decider.Decider[UserState]{
		Initial: UserState{},
		Apply:   applyUserEvents,
	}

	userRepo, err := decider.NewRepository(store, bus, userDecider)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	handler := func(ctx context.Context, cmd command.Command) error {
		c, ok := cmd.(*CreateUser)
		if !ok {
			return command.ErrTypeAssertion
		}

		return userRepo.Execute(
			ctx, c.StreamID(), "User",
			func(_ UserState, version event.Version) ([]event.Event, error) {
				evt, evtErr := event.NewEvent(
					"UserCreated", c.StreamID(), "User", version.Add(1), []byte(c.Name),
				)
				if evtErr != nil {
					return nil, evtErr
				}

				return []event.Event{evt}, nil
			},
		)
	}

	if err := cmdDispatcher.Register("CreateUser", handler); err != nil {
		t.Fatalf("register: %v", err)
	}

	_ = bus.SubscribeAll(func(_ context.Context, _ event.Event) error { return nil })

	aggID := id.NewAggregateID()
	createCmd, _ := command.New("CreateUser", aggID)
	cmd := &CreateUser{BasicCommand: createCmd, Name: "Bob"}

	if err := cmdDispatcher.Dispatch(context.Background(), cmd); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	tp.ForceFlush(context.Background())
	spans := exporter.GetSpans()

	if len(spans) < 3 {
		names := readOnlySpanNames(spans)
		t.Fatalf("expected at least 3 spans, got %d: %v", len(spans), names)
	}

	// All spans must share one trace ID.
	traceID := spans[0].SpanContext.TraceID()

	for _, s := range spans[1:] {
		if s.SpanContext.TraceID() != traceID {
			t.Errorf("span %q has trace ID %s, expected %s",
				s.Name, s.SpanContext.TraceID(), traceID)
		}
	}

	// Expected span names must be present.
	names := readOnlySpanNames(spans)
	expected := []string{"command.handle", "decider.execute", "decider.load"}

	for _, want := range expected {
		if !slices.Contains(names, want) {
			t.Errorf("expected span %q in tree, got: %v", want, names)
		}
	}

	// Verify parent-child chain.
	parentMap := buildParentMap(spans)

	if parent, ok := parentMap["decider.execute"]; !ok || parent != "command.handle" {
		t.Errorf("expected decider.execute parent = command.handle, got %q", parent)
	}

	if parent, ok := parentMap["decider.load"]; !ok || parent != "decider.execute" {
		t.Errorf("expected decider.load parent = decider.execute, got %q", parent)
	}

	// Golden: the span names set is stable for regression detection. The exact
	// set may grow as more layers are instrumented, but these core spans must
	// always be present.
	sortedNames := append([]string(nil), names...)
	sort.Strings(sortedNames)

	t.Logf("span tree: %v", sortedNames)
}

func readOnlySpanNames(spans []tracetest.SpanStub) []string {
	names := make([]string, 0, len(spans))
	for _, s := range spans {
		names = append(names, s.Name)
	}

	return names
}

// buildParentMap maps each span name to its parent span name.
func buildParentMap(spans []tracetest.SpanStub) map[string]string {
	result := make(map[string]string, len(spans))

	byID := make(map[trace.SpanID]string, len(spans))
	for _, s := range spans {
		byID[s.SpanContext.SpanID()] = s.Name
	}

	for _, s := range spans {
		if s.Parent.SpanID().IsValid() {
			if parentName, ok := byID[s.Parent.SpanID()]; ok {
				result[s.Name] = parentName
			}
		}
	}

	return result
}
