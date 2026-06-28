package middleware

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// BenchmarkOTelBundle_CommandOverhead measures the per-command cost of the
// OTel bundle (tracing + metrics) vs a bare dispatcher. Uses a no-op SDK
// provider to measure instrumentation overhead, not exporter cost.
func BenchmarkOTelBundle_CommandOverhead(b *testing.B) {
	tp := sdktrace.NewTracerProvider()
	defer tp.Shutdown(context.Background())

	origTP := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(origTP)

	bundle, err := NewOTelBundle(
		tp.Tracer("bench"),
		otel.GetMeterProvider().Meter("bench"),
	)
	if err != nil {
		b.Fatalf("new bundle: %v", err)
	}

	disp := command.NewDispatcher()
	disp.Use(bundle.Command()...)

	handler := disp.Register("bench.cmd", func(_ context.Context, _ command.Command) error {
		return nil
	})
	if handler != nil {
		b.Fatal("unexpected error registering handler")
	}

	aggID := id.NewAggregateID()
	cmd, _ := command.New("bench.cmd", aggID)

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = disp.Dispatch(ctx, cmd)
	}
}

// BenchmarkOTelBundle_TracingOnly measures the cost with metrics disabled.
func BenchmarkOTelBundle_TracingOnly(b *testing.B) {
	tp := sdktrace.NewTracerProvider()
	defer tp.Shutdown(context.Background())

	origTP := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(origTP)

	bundle, err := NewOTelBundle(
		tp.Tracer("bench"), nil,
		WithMetricsDisabled(),
	)
	if err != nil {
		b.Fatalf("new bundle: %v", err)
	}

	disp := command.NewDispatcher()
	disp.Use(bundle.Command()...)

	_ = disp.Register("bench.cmd", func(_ context.Context, _ command.Command) error {
		return nil
	})

	aggID := id.NewAggregateID()
	cmd, _ := command.New("bench.cmd", aggID)

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = disp.Dispatch(ctx, cmd)
	}
}
