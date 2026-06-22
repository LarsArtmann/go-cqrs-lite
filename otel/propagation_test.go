package otel_test

import (
	"context"
	"testing"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
)

func TestCorrelationID_BaggageRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	corrID := "abc-123-def"

	ctx = cqrsotel.WithCorrelationID(ctx, corrID)

	got := cqrsotel.CorrelationIDFromContext(ctx)
	if got != corrID {
		t.Errorf("CorrelationIDFromContext = %q, want %q", got, corrID)
	}
}

func TestCorrelationID_EmptyWhenNotSet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	got := cqrsotel.CorrelationIDFromContext(ctx)
	if got != "" {
		t.Errorf("CorrelationIDFromContext = %q, want empty", got)
	}
}

func TestCorrelationID_BaggagePropagation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	corrID := "trace-456"

	// Set correlation ID in context A
	ctxA := cqrsotel.WithCorrelationID(ctx, corrID)

	// Simulate cross-service propagation via headers
	carrier := make(map[string]string)
	propagator := cqrsotel.NewTextMapPropagator()
	propagator.Inject(ctxA, propagationMapCarrier{carrier})

	// Extract in context B
	ctxB := context.Background()
	ctxB = propagator.Extract(ctxB, propagationMapCarrier{carrier})

	got := cqrsotel.CorrelationIDFromContext(ctxB)
	if got != corrID {
		t.Errorf("after propagation: CorrelationIDFromContext = %q, want %q", got, corrID)
	}
}

type propagationMapCarrier struct {
	m map[string]string
}

func (c propagationMapCarrier) Get(key string) string { return c.m[key] }

func (c propagationMapCarrier) Set(key, val string) { c.m[key] = val }

func (c propagationMapCarrier) Keys() []string {
	keys := make([]string, 0, len(c.m))

	for k := range c.m {
		keys = append(keys, k)
	}

	return keys
}

func TestNewCQRSViews_NotNil(t *testing.T) {
	t.Parallel()

	views := cqrsotel.NewCQRSViews()
	if len(views) == 0 {
		t.Fatal("NewCQRSViews returned empty slice")
	}
}
