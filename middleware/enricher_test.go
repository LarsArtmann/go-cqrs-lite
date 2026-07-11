package middleware

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

func TestOTelCorrelationEnricher_WithBaggage(t *testing.T) {
	t.Parallel()

	ctx := cqrsotel.WithCorrelationID(context.Background(), "trace-abc-123")

	opts := OTelCorrelationEnricher(ctx)

	if len(opts) != 1 {
		t.Fatalf("expected 1 option, got %d", len(opts))
	}

	evt := eventtest.NewEvent(t, "test.event", id.NewAggregateID(), "Test", 1, nil)
	opts[0](evt)

	got := OTelCorrelationIDFromEvent(evt)
	if got != "trace-abc-123" {
		t.Errorf("expected 'trace-abc-123', got %q", got)
	}
}

func TestOTelCorrelationEnricher_NoBaggage(t *testing.T) {
	t.Parallel()

	opts := OTelCorrelationEnricher(context.Background())

	if opts != nil {
		t.Fatalf("expected nil options when no correlation ID in context, got %d", len(opts))
	}
}

func TestOTelCorrelationEnricher_AcceptsArbitraryStrings(t *testing.T) {
	t.Parallel()

	traceID := "4bf92f3577b34da6a3ce929d0e0e4736"
	ctx := cqrsotel.WithCorrelationID(context.Background(), traceID)

	evt := eventtest.NewEvent(t, "test.event", id.NewAggregateID(), "Test", 1, nil)

	for _, opt := range OTelCorrelationEnricher(ctx) {
		opt(evt)
	}

	got := OTelCorrelationIDFromEvent(evt)
	if got != traceID {
		t.Errorf("expected %q, got %q", traceID, got)
	}
}

func TestOTelCorrelationEnricher_ComposesWithCommandCausality(t *testing.T) {
	t.Parallel()

	cmdID := id.NewCommandID()
	ctx := cqrsotel.WithCorrelationID(context.Background(), "dist-trace-001")
	ctx = event.WithCommandCausality(ctx, "user.create", cmdID)

	composite := event.CompositeEnricher(
		event.CommandCausalityEnricher,
		OTelCorrelationEnricher,
	)

	opts := composite(ctx)

	if len(opts) != 4 {
		t.Fatalf(
			"expected 4 options (causation + command type + command ID + otel corr), got %d",
			len(opts),
		)
	}

	evt := eventtest.NewEvent(t, "user.created", id.NewAggregateID(), "User", 1, nil)

	for _, opt := range opts {
		opt(evt)
	}

	otelCorr := OTelCorrelationIDFromEvent(evt)
	if otelCorr != "dist-trace-001" {
		t.Errorf("otel correlation: expected 'dist-trace-001', got %q", otelCorr)
	}

	cmdType := evt.Metadata().Custom[event.MetadataKeyCommandType]
	if cmdType != "user.create" {
		t.Errorf("command type: expected 'user.create', got %q", cmdType)
	}
}

func TestOTelCorrelationIDFromEvent_NotSet(t *testing.T) {
	t.Parallel()

	evt := eventtest.NewEvent(t, "test.event", id.NewAggregateID(), "Test", 1, nil)

	got := OTelCorrelationIDFromEvent(evt)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}
