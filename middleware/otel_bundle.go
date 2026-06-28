package middleware

import (
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
	tracer   cqrsotel.Tracer
	recorder *OTelMetricsRecorder
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
func NewOTelBundle(tracer cqrsotel.Tracer, meter cqrsotel.Meter) (*OTelBundle, error) {
	recorder, err := NewOTelMetricsRecorder(meter)
	if err != nil {
		return nil, fmt.Errorf("create otel metrics recorder: %w", err)
	}

	return &OTelBundle{tracer: tracer, recorder: recorder}, nil
}

// Command returns the recommended OTel middleware chain for command handlers:
// tracing (server span) then metrics (duration + count).
func (b *OTelBundle) Command() []command.Middleware {
	return []command.Middleware{
		CommandTracing(b.tracer),
		CommandOTelMetricsWithCounter(b.recorder.Histogram(), b.recorder.Counter()),
	}
}

// Event returns the recommended OTel middleware chain for event handlers:
// tracing (consumer span) then metrics (duration + count).
func (b *OTelBundle) Event() []event.Middleware {
	return []event.Middleware{
		EventTracing(b.tracer),
		EventOTelMetricsWithCounter(b.recorder.Histogram(), b.recorder.Counter()),
	}
}

// Query returns the recommended OTel middleware chain for query handlers:
// tracing (server span) then metrics (duration + count).
func (b *OTelBundle) Query() []query.Middleware {
	return []query.Middleware{
		QueryTracing(b.tracer),
		QueryOTelMetricsWithCounter(b.recorder.Histogram(), b.recorder.Counter()),
	}
}

// Publish returns the recommended OTel publish middleware for the event bus:
// tracing (producer span) for the publish operation.
func (b *OTelBundle) Publish() []event.PublishMiddleware {
	return []event.PublishMiddleware{
		EventPublishTracing(b.tracer),
	}
}

// Recorder returns the underlying OTelMetricsRecorder, useful for custom
// middleware that needs access to the same instruments.
func (b *OTelBundle) Recorder() *OTelMetricsRecorder {
	return b.recorder
}

// Tracer returns the tracer used by this bundle.
func (b *OTelBundle) Tracer() cqrsotel.Tracer {
	return b.tracer
}
