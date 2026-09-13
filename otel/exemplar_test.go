package otel_test

import (
	"context"
	"testing"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"

	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// Exemplars are ON by default: the SDK ships the trace-based exemplar
// filter, so any histogram observation recorded under a sampled span
// carries the trace/span IDs into the metric stream — the bridge between
// cqrs.* dashboards and the traces behind a latency spike. No Setup option
// is needed; OTEL_METRICS_EXEMPLAR_FILTER=always_off disables it.
func TestSetup_ExemplarsFlowFromSampledSpans(t *testing.T) {
	t.Parallel()

	reader := sdkmetric.NewManualReader()
	provider, err := cqrsotel.Setup(
		cqrsotel.WithMetricReader(reader),
		cqrsotel.WithoutGlobalRegistration(),
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

	ctx := context.Background()

	tracer := provider.AsTracerProvider().Tracer("cqrs/exemplar-test")
	ctx, span := tracer.Start(ctx, "cqrs.test.operation")

	hist, err := provider.AsMeterProvider().
		Meter("cqrs/test").
		Float64Histogram("cqrs.test.duration")
	if err != nil {
		t.Fatal(err)
	}

	hist.Record(ctx, 12.5)

	span.SetStatus(codes.Ok, "")
	span.End()

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			histData, ok := m.Data.(metricdata.Histogram[float64])
			if !ok {
				continue
			}

			for _, dp := range histData.DataPoints {
				if len(dp.Exemplars) == 0 {
					continue
				}

				if len(dp.Exemplars[0].SpanID) > 0 && len(dp.Exemplars[0].TraceID) > 0 {
					return // proven: exemplar carries the sampled span context
				}
			}
		}
	}

	t.Fatal("no exemplar with valid trace/span IDs collected — exemplars are not flowing")
}
