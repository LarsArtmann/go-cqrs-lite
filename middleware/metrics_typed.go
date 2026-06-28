package middleware

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

// TypedMetricsRecorder records handler execution metrics using typed
// OpenTelemetry attributes, replacing the error-prone alternating
// key-value string pairs of MetricsRecorder. Prefer this interface for new
// code: it makes malformed label pairs unrepresentable.
type TypedMetricsRecorder interface {
	ObserveTyped(ctx context.Context, operation string, duration time.Duration, attrs ...cqrsotel.KeyValue)
}

// typeAttrFor maps a message kind to its canonical OTel attribute key.
func typeAttrFor(kind string) string {
	switch kind {
	case kindCommand:
		return cqrsotel.AttrCommandType
	case kindEvent:
		return cqrsotel.AttrEventType
	case kindQuery:
		return cqrsotel.AttrQueryType
	default:
		return "cqrs.type"
	}
}

// NewTypedMetrics returns a generic middleware that records handler metrics
// via a TypedMetricsRecorder. The operation name is the message kind and
// attributes include message kind, message type, and status (success/error).
func NewTypedMetrics[M any](adapter MessageAdapter[M], recorder TypedMetricsRecorder) Middleware[M] {
	return func(next Handler[M]) Handler[M] {
		return func(ctx context.Context, msg M) error {
			start := time.Now()
			err := next(ctx, msg)

			status := cqrsotel.StatusSuccess
			if err != nil {
				status = cqrsotel.StatusError
			}

			recorder.ObserveTyped(
				ctx, adapter.Kind, time.Since(start),
				cqrsotel.AttrString(cqrsotel.AttrMessageKind, adapter.Kind),
				cqrsotel.AttrString(typeAttrFor(adapter.Kind), adapter.ExtractType(msg)),
				cqrsotel.AttrString(cqrsotel.AttrStatus, status),
			)

			return err
		}
	}
}

// CommandTypedMetrics returns a command middleware that records metrics via a
// TypedMetricsRecorder with typed attributes.
func CommandTypedMetrics(recorder TypedMetricsRecorder) command.Middleware {
	return AsCommand(NewTypedMetrics(CommandAdapter, recorder))
}

// EventTypedMetrics returns an event middleware that records metrics via a
// TypedMetricsRecorder with typed attributes.
func EventTypedMetrics(recorder TypedMetricsRecorder) event.Middleware {
	return AsEvent(NewTypedMetrics(EventAdapter, recorder))
}

// QueryTypedMetrics returns a query middleware that records metrics via a
// TypedMetricsRecorder with typed attributes.
func QueryTypedMetrics(recorder TypedMetricsRecorder) query.Middleware {
	return AsQuery(NewTypedMetrics(QueryAdapter, recorder))
}
