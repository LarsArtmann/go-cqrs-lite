package otel

import (
	"context"
	"sync"
	"testing"

	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// recordingProcessor records the sequence of lifecycle calls so tests can pin
// the order Provider.Shutdown flushes vs shuts down.
type recordingProcessor struct {
	mu     sync.Mutex
	events []string
}

func (p *recordingProcessor) OnStart(context.Context, sdktrace.ReadWriteSpan) {}

func (p *recordingProcessor) OnEnd(sdktrace.ReadOnlySpan) {}

func (p *recordingProcessor) Shutdown(context.Context) error {
	p.record("shutdown")

	return nil
}

func (p *recordingProcessor) ForceFlush(context.Context) error {
	p.record("flush")

	return nil
}

func (p *recordingProcessor) record(event string) {
	p.mu.Lock()
	p.events = append(p.events, event)
	p.mu.Unlock()
}

func (p *recordingProcessor) snapshot() []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	return append([]string(nil), p.events...)
}

// TestProvider_Shutdown_FlushesBeforeShutdown pins the flush-before-shutdown
// ordering: without the explicit ForceFlush, spans sitting in the batcher
// (batch interval not elapsed) race against process exit.
func TestProvider_Shutdown_FlushesBeforeShutdown(t *testing.T) {
	t.Parallel()

	proc := &recordingProcessor{}

	provider, err := Setup(
		WithSpanProcessor(proc),
		WithMetricReader(metric.NewManualReader()),
		WithoutGlobalRegistration(),
	)
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}

	if err := provider.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	events := proc.snapshot()

	if len(events) != 2 || events[0] != "flush" || events[1] != "shutdown" {
		t.Fatalf("expected [flush shutdown], got %v", events)
	}
}
