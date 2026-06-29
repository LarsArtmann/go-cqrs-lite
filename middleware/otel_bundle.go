package middleware

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

// OTelBundle is a pre-wired set of OpenTelemetry middleware for all three
// CQRS message kinds (command, event, query) plus the event publish path.
// It eliminates the boilerplate of individually wiring tracing and metrics
// middleware for each dispatcher and bus.
//
// Create one via NewOTelBundle, then spread the returned slices into your
// dispatchers and bus:
//
//	bundle, err := middleware.NewOTelBundle(
//	    cqrsotel.NewTracer("app"),
//	    cqrsotel.NewMeter("app"),
//	)
//	if err != nil {
//	    return err
//	}
//	cmdDisp.Use(bundle.Command()...)
//	bus.Use(bundle.Event()...)
//	qryDisp.Use(bundle.Query()...)
//	bus.UsePublish(bundle.Publish()...)
//
// Each method returns the recommended middleware ordering: tracing first
// (so the span wraps the entire operation including metrics), then metrics.
type OTelBundle struct {
	tracer         cqrsotel.Tracer
	recorder       *OTelMetricsRecorder
	metricsEnabled bool
}

// BundleOption configures the OTel bundle.
type BundleOption func(*bundleConfig)

type bundleConfig struct {
	metricsEnabled bool
}

// WithMetricsDisabled produces a tracing-only bundle: the Command/Event/Query
// chains emit spans but skip metric recording. The meter argument to
// NewOTelBundle may be nil when this option is set.
func WithMetricsDisabled() BundleOption {
	return func(c *bundleConfig) {
		c.metricsEnabled = false
	}
}

// NewOTelBundle creates a complete OTel middleware bundle from a tracer and
// meter. The meter is used to create the standard CQRS instruments
// (cqrs.operation.duration histogram + cqrs.operation.count counter).
//
// Both arguments are typically obtained from the global providers:
//
//	bundle, _ := middleware.NewOTelBundle(
//	    cqrsotel.NewTracer("orders"),
//	    cqrsotel.NewMeter("orders"),
//	)
//
// Or from an explicit provider for testing:
//
//	bundle, _ := middleware.NewOTelBundle(
//	    provider.Tracer("orders"),
//	    provider.Meter("orders"),
//	)
//
// For a tracing-only setup (no metrics), pass a nil meter with
// WithMetricsDisabled:
//
//	bundle, _ := middleware.NewOTelBundle(
//	    cqrsotel.NewTracer("orders"), nil,
//	    middleware.WithMetricsDisabled(),
//	)
func NewOTelBundle(
	tracer cqrsotel.Tracer,
	meter cqrsotel.Meter,
	opts ...BundleOption,
) (*OTelBundle, error) {
	cfg := &bundleConfig{
		metricsEnabled: true,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	b := &OTelBundle{ //nolint:exhaustruct // recorder set conditionally below
		tracer:         tracer,
		metricsEnabled: cfg.metricsEnabled,
	}

	if cfg.metricsEnabled {
		if meter == nil {
			return nil, ErrMeterRequired
		}

		recorder, err := NewOTelMetricsRecorder(meter)
		if err != nil {
			return nil, fmt.Errorf("create otel metrics recorder: %w", err)
		}

		b.recorder = recorder
	}

	return b, nil
}

// Command returns the recommended OTel middleware chain for command handlers:
// tracing (server span) then metrics (duration + count, unless disabled).
func (b *OTelBundle) Command() []command.Middleware {
	chain := []command.Middleware{
		CommandTracing(b.tracer),
	}

	if b.metricsEnabled {
		chain = append(chain,
			CommandOTelMetricsWithCounter(b.recorder.Histogram(), b.recorder.Counter()))
	}

	return chain
}

// Event returns the recommended OTel middleware chain for event handlers:
// tracing (consumer span) then metrics (duration + count, unless disabled).
func (b *OTelBundle) Event() []event.Middleware {
	chain := []event.Middleware{
		EventTracing(b.tracer),
	}

	if b.metricsEnabled {
		chain = append(chain,
			EventOTelMetricsWithCounter(b.recorder.Histogram(), b.recorder.Counter()))
	}

	return chain
}

// Query returns the recommended OTel middleware chain for query handlers:
// tracing (server span) then metrics (duration + count, unless disabled).
func (b *OTelBundle) Query() []query.Middleware {
	chain := []query.Middleware{
		QueryTracing(b.tracer),
	}

	if b.metricsEnabled {
		chain = append(chain,
			QueryOTelMetricsWithCounter(b.recorder.Histogram(), b.recorder.Counter()))
	}

	return chain
}

// Publish returns the recommended OTel publish middleware for the event bus:
// tracing (producer span) for the publish operation.
func (b *OTelBundle) Publish() []event.PublishMiddleware {
	return []event.PublishMiddleware{
		EventPublishTracing(b.tracer),
	}
}

// CorrelationEnricher returns a decider event enricher that bridges OTel
// baggage correlation IDs into event metadata. Wire it via
// decider.WithEnricher so correlation IDs flow automatically through every
// event produced by the aggregate:
//
//	repo, _ := decider.NewRepository(store, bus, decider.WithEnricher(
//	    bundle.CorrelationEnricher()))
func (b *OTelBundle) CorrelationEnricher() func(context.Context) []event.Option {
	return OTelCorrelationEnricher
}

// Recorder returns the underlying OTelMetricsRecorder, useful for custom
// middleware that needs access to the same instruments. Returns nil when
// metrics are disabled.
func (b *OTelBundle) Recorder() *OTelMetricsRecorder {
	return b.recorder
}

// Tracer returns the tracer used by this bundle.
func (b *OTelBundle) Tracer() cqrsotel.Tracer {
	return b.tracer
}
