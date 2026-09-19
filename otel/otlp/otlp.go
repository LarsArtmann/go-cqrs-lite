// Package otlp wires OpenTelemetry's OTLP/HTTP exporters into
// cqrsotel.Setup in one call — the common "ship traces and metrics to a
// collector" path without assembling exporters by hand:
//
//	provider, err := cqrsotlp.SetupOTLP(ctx, cqrsotlp.OTLPConfig{
//		Endpoint:    "localhost:4318",
//		ServiceName: "my-service",
//	})
//	defer func() { _ = provider.Shutdown(ctx) }()
//
// Transport is OTLP/HTTP, so no gRPC dependency enters the module graph;
// gRPC users keep injecting their own exporter via
// cqrsotel.WithSpanExporter. Everything cqrsotel.Setup provides — CQRS
// histogram views, propagation, resource attributes, flush-on-shutdown —
// applies unchanged.
package otlp

import (
	"context"

	otlpmetrichttp "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otlptracehttp "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// OTLPConfig configures the OTLP/HTTP exporters behind SetupOTLP.
type OTLPConfig struct {
	// Endpoint is the collector's host:port without scheme, e.g.
	// "localhost:4318" or "otel-collector.observability:4318".
	Endpoint string

	// Insecure switches the exporters from HTTPS to plain HTTP — the usual
	// choice for a local or in-cluster sidecar collector.
	Insecure bool

	// Headers are sent with every export request (collector auth).
	Headers map[string]string

	// ServiceName, ServiceVersion, and InstanceID feed the OTel resource
	// attributes alongside the standard SDK detectors.
	ServiceName    string
	ServiceVersion string
	InstanceID     string
}

// SetupOTLP builds OTLP/HTTP trace and metric exporters for cfg and hands
// them to cqrsotel.Setup. Extra opts run after the OTLP wiring, so any
// cqrsotel.SetupOption can override it (custom span processors, a different
// propagator, WithoutGlobalRegistration for tests).
func SetupOTLP(
	ctx context.Context,
	cfg OTLPConfig,
	opts ...cqrsotel.SetupOption,
) (*cqrsotel.Provider, error) {
	spanExporter, err := otlptracehttp.New(ctx, traceOptions(cfg)...)
	if err != nil {
		return nil, err //nolint:wrapcheck // exporter-scoped SDK error
	}

	metricExporter, err := otlpmetrichttp.New(ctx, metricOptions(cfg)...)
	if err != nil {
		return nil, err //nolint:wrapcheck // exporter-scoped SDK error
	}

	// The exporter ships payloads; the periodic reader owns the export
	// cadence and makes it a metric.Reader for cqrsotel.Setup.
	metricReader := sdkmetric.NewPeriodicReader(metricExporter)

	setupOpts := make([]cqrsotel.SetupOption, 0, 3+len(opts))
	setupOpts = append(setupOpts,
		cqrsotel.WithSpanExporter(spanExporter),
		cqrsotel.WithMetricReader(metricReader),
		cqrsotel.WithService(cfg.ServiceName, cfg.ServiceVersion, cfg.InstanceID),
	)

	//nolint:contextcheck // resource metadata is process-scoped, not caller-ctx-scoped
	return cqrsotel.Setup(
		append(setupOpts, opts...)...)
}

func traceOptions(cfg OTLPConfig) []otlptracehttp.Option {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.Endpoint),
	}

	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	if len(cfg.Headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(cfg.Headers))
	}

	return opts
}

func metricOptions(cfg OTLPConfig) []otlpmetrichttp.Option {
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(cfg.Endpoint),
	}

	if cfg.Insecure {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}

	if len(cfg.Headers) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(cfg.Headers))
	}

	return opts
}
