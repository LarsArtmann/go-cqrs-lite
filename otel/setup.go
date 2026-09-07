//go:build !js

package otel

import (
	"context"
	"errors"
	"fmt"
	"io"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// SetupOption configures the provider setup.
type SetupOption func(*setupConfig)

type setupConfig struct {
	serviceName    string
	serviceVersion string
	instanceID     string
	spanExporter   sdktrace.SpanExporter
	metricReader   metric.Reader
	propagator     propagation.TextMapPropagator
	stdoutWriter   io.Writer // non-nil → construct a pretty-printing stdout span exporter
	// spanProcessors are appended as raw span processors (after any exporter
	// batcher). Set by WithSpanProcessor for custom pipelines and tests.
	spanProcessors []sdktrace.SpanProcessor
	// skipGlobalRegistration omits registering the providers as the process-wide
	// globals. Set by WithoutGlobalRegistration for isolated use (tests,
	// multi-service processes). The returned Provider is fully functional.
	skipGlobalRegistration bool
}

// WithSpanProcessor attaches a raw span processor to the tracer provider
// (after any exporter batcher). Useful for custom pipelines and tests that
// need to observe provider lifecycle events.
func WithSpanProcessor(p sdktrace.SpanProcessor) SetupOption {
	return func(c *setupConfig) {
		c.spanProcessors = append(c.spanProcessors, p)
	}
}

// WithService identifies the service in telemetry via resource attributes.
// serviceName is required for meaningful traces; version and instanceID are optional.
func WithService(name, version, instanceID string) SetupOption {
	return func(c *setupConfig) {
		c.serviceName = name
		c.serviceVersion = version
		c.instanceID = instanceID
	}
}

// WithSpanExporter attaches a span exporter (OTLP, stdout, Jaeger, etc.).
// Without one, spans are recorded but not exported — useful for in-memory testing.
func WithSpanExporter(e sdktrace.SpanExporter) SetupOption {
	return func(c *setupConfig) {
		c.spanExporter = e
	}
}

// WithMetricReader attaches a metric reader (OTLP, prometheus, stdout, etc.).
// When omitted, no metric reader is configured.
func WithMetricReader(r metric.Reader) SetupOption {
	return func(c *setupConfig) {
		c.metricReader = r
	}
}

// WithPropagator overrides the default W3C (trace-context + baggage) propagator.
func WithPropagator(p propagation.TextMapPropagator) SetupOption {
	return func(c *setupConfig) {
		c.propagator = p
	}
}

// WithStdoutExporter prints spans to the given writer with pretty-printing.
// Ideal for local development and debugging — pass os.Stdout to see traces
// in your terminal. The exporter is created internally; for a custom stdout
// configuration use WithSpanExporter with a manually constructed exporter.
func WithStdoutExporter(w io.Writer) SetupOption {
	return func(c *setupConfig) {
		c.stdoutWriter = w
	}
}

// WithoutGlobalRegistration skips registering the providers as the process-wide
// global TracerProvider, MeterProvider, and TextMapPropagator. Use this when you
// need an isolated Provider — e.g. in tests, or when running multiple services
// in one process where each owns its providers. The returned Provider is fully
// functional; only otel.SetTracerProvider / SetMeterProvider / SetTextMapPropagator
// are skipped, so otel.GetTracerProvider() is left unchanged.
func WithoutGlobalRegistration() SetupOption {
	return func(c *setupConfig) {
		c.skipGlobalRegistration = true
	}
}

// Provider wraps TracerProvider and MeterProvider with a unified Shutdown.
type Provider struct {
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *metric.MeterProvider
}

// AsTracerProvider returns the underlying OTel TracerProvider.
func (p *Provider) AsTracerProvider() *sdktrace.TracerProvider {
	return p.tracerProvider
}

// AsMeterProvider returns the underlying OTel MeterProvider.
func (p *Provider) AsMeterProvider() *metric.MeterProvider {
	return p.meterProvider
}

// errShutdown indicates one or more providers failed to shut down cleanly.
var errShutdown = errors.New("otel provider shutdown incomplete")

var errBuildResource = errors.New("failed to build OTel resource")

// Shutdown flushes pending spans and metrics, then releases resources.
// Always call this on application exit.
func (p *Provider) Shutdown(ctx context.Context) error {
	var errs []error

	// ForceFlush BEFORE Shutdown: the batcher drops spans recorded between the
	// last flush and Shutdown only if the batch interval has not elapsed, and
	// ForceFlush gives the exporter a bounded window to drain batches while
	// the provider is still fully active. Ordering (flush → shutdown) is
	// pinned by TestProvider_Shutdown_FlushesBeforeShutdown.
	if err := p.tracerProvider.ForceFlush(ctx); err != nil {
		errs = append(errs, fmt.Errorf("tracer flush: %w", err))
	}

	if err := p.tracerProvider.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("tracer shutdown: %w", err))
	}

	if err := p.meterProvider.ForceFlush(ctx); err != nil {
		errs = append(errs, fmt.Errorf("meter flush: %w", err))
	}

	if err := p.meterProvider.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("meter shutdown: %w", err))
	}

	if len(errs) > 0 {
		return errors.Join(append([]error{errShutdown}, errs...)...)
	}

	return nil
}

// Setup creates and registers TracerProvider and MeterProvider in one call.
// It configures the W3C propagator, CQRS-optimized histogram views, and a
// resource identifying the service. The returned Provider owns both providers.
//
// Without a span exporter, spans are recorded but not exported — ideal for
// in-memory testing or when an exporter will be attached later. Attach a
// real exporter via WithSpanExporter for production.
//
// Typical usage:
//
//	provider, err := cqrsotel.Setup(
//	    cqrsotel.WithService("orders", "1.0.0", "instance-1"),
//	    cqrsotel.WithSpanExporter(otlpExporter),
//	)
//	if err != nil {
//	    return err
//	}
//	defer provider.Shutdown(ctx)
//
// The global TracerProvider and MeterProvider are set automatically, so
// cqrsotel.NewTracer("middleware") and cqrsotel.NewMeter("middleware")
// resolve to these providers without additional wiring.
func Setup(opts ...SetupOption) (*Provider, error) {
	cfg := &setupConfig{} //nolint:exhaustruct_v5 // options applied below

	for _, opt := range opts {
		opt(cfg)
	}

	res, err := buildResource(cfg)
	if err != nil {
		return nil, err
	}

	spanExporter := cfg.spanExporter
	if spanExporter == nil && cfg.stdoutWriter != nil {
		spanExporter, err = stdouttrace.New(
			stdouttrace.WithWriter(cfg.stdoutWriter),
			stdouttrace.WithPrettyPrint(),
		)
		if err != nil {
			return nil, fmt.Errorf("stdout exporter: %w", err)
		}
	}

	propagator := cfg.propagator
	if propagator == nil {
		propagator = NewTextMapPropagator()
	}

	if !cfg.skipGlobalRegistration {
		otel.SetTextMapPropagator(propagator)
	}

	tpOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
	}

	if spanExporter != nil {
		tpOpts = append(tpOpts, sdktrace.WithBatcher(spanExporter))
	}

	for _, proc := range cfg.spanProcessors {
		tpOpts = append(tpOpts, sdktrace.WithSpanProcessor(proc))
	}

	tracerProvider := sdktrace.NewTracerProvider(tpOpts...)

	mpOpts := []metric.Option{
		metric.WithResource(res),
		metric.WithView(NewCQRSViews()...),
	}

	if cfg.metricReader != nil {
		mpOpts = append(mpOpts, metric.WithReader(cfg.metricReader))
	}

	meterProvider := metric.NewMeterProvider(mpOpts...)

	if !cfg.skipGlobalRegistration {
		otel.SetTracerProvider(tracerProvider)
		otel.SetMeterProvider(meterProvider)
	}

	return &Provider{tracerProvider: tracerProvider, meterProvider: meterProvider}, nil
}

func buildResource(cfg *setupConfig) (*resource.Resource, error) {
	attrs := ServiceResourceAttributes(cfg.serviceName, cfg.serviceVersion, cfg.instanceID)

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(attrs...),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errBuildResource, err)
	}

	return res, nil
}
