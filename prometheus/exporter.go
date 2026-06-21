package prometheus

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

// Option configures the Prometheus setup.
type Option func(*config)

type config struct {
	registry    prometheus.Registerer
	handlerOpts promhttp.HandlerOpts
}

// WithRegistry uses a custom Prometheus registry instead of the default.
func WithRegistry(r prometheus.Registerer) Option {
	return func(c *config) {
		c.registry = r
	}
}

// WithHandlerOptions configures the promhttp.HandlerOpts (e.g., timeouts,
// error handling, enable gzip).
func WithHandlerOptions(opts promhttp.HandlerOpts) Option {
	return func(c *config) {
		c.handlerOpts = opts
	}
}

// Provider wraps a MeterProvider with a Shutdown method. The underlying
// MeterProvider is accessible via AsMeterProvider() for use with
// otel.SetMeterProvider().
type Provider struct {
	meterProvider *metric.MeterProvider
	handler       http.Handler
}

// AsMeterProvider returns the underlying OTel MeterProvider.
// Pass this to otel.SetMeterProvider().
func (p *Provider) AsMeterProvider() *metric.MeterProvider {
	return p.meterProvider
}

// Handler returns the HTTP handler for the /metrics endpoint.
func (p *Provider) Handler() http.Handler {
	return p.handler
}

// Shutdown flushes pending metrics and releases resources.
func (p *Provider) Shutdown(ctx context.Context) error {
	return fmt.Errorf("shutdown: %w", p.meterProvider.Shutdown(ctx))
}

// Setup creates a Prometheus-backed MeterProvider and HTTP handler in one call.
// The returned Provider exposes OTel instruments as Prometheus metrics.
//
// Typical usage:
//
//	provider, err := prometheus.Setup()
//	if err != nil { return err }
//	defer provider.Shutdown(context.Background())
//
//	otel.SetMeterProvider(provider.AsMeterProvider())
//	mux.Handle("/metrics", provider.Handler())
func Setup(opts ...Option) (*Provider, error) {
	cfg := &config{} //nolint:exhaustruct // options applied below

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.registry == nil {
		cfg.registry = prometheus.NewRegistry()
	}

	exporter, err := otelprom.New(
		otelprom.WithRegisterer(cfg.registry),
	)
	if err != nil {
		return nil, fmt.Errorf("create prometheus exporter: %w", err)
	}

	meterProvider := metric.NewMeterProvider(metric.WithReader(exporter))

	gatherer, ok := cfg.registry.(prometheus.Gatherer)
	if !ok {
		return nil, errNotGatherer
	}

	handler := promhttp.HandlerFor(
		gatherer,
		cfg.handlerOpts,
	)

	return &Provider{
		meterProvider: meterProvider,
		handler:       handler,
	}, nil
}

var errNotGatherer = errors.New("registry does not implement prometheus.Gatherer")
