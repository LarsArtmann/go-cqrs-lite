package http

import (
	"context"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// ReplayMetrics holds the OpenTelemetry instruments for SSE replay
// observability. A zero-value ReplayMetrics is a no-op: all methods are
// nil-safe, so callers that opt out of metrics pay nothing.
//
// Use NewReplayMetrics to construct a usable instance from a [cqrsotel.Meter].
type ReplayMetrics struct {
	duration        cqrsotel.Float64Histogram // cqrs.sse.replay.duration (ms)
	eventsCounter   cqrsotel.Int64Counter     // cqrs.sse.replay.events
	incompleteCount cqrsotel.Int64Counter     // cqrs.sse.replay.incomplete
}

// NewReplayMetrics creates a ReplayMetrics wired to the given meter.
// Returns nil if meter is nil — callers store the nil and the no-op methods
// handle it gracefully.
func NewReplayMetrics(meter cqrsotel.Meter) (*ReplayMetrics, error) {
	if meter == nil {
		return nil, nil //nolint:nilnil // nil meter = opt-out, nil metrics is the correct sentinel
	}

	duration, err := meter.Float64Histogram(
		"cqrs.sse.replay.duration",
		cqrsotel.MetricWithDescription("Duration of SSE journal replay"),
		cqrsotel.MetricWithUnit("ms"),
	)
	if err != nil {
		return nil, err //nolint:wrapcheck // otel SDK error
	}

	events, err := meter.Int64Counter(
		"cqrs.sse.replay.events",
		cqrsotel.CounterMetricWithDescription("Total events delivered during SSE replay"),
	)
	if err != nil {
		return nil, err //nolint:wrapcheck // otel SDK error
	}

	incomplete, err := meter.Int64Counter(
		"cqrs.sse.replay.incomplete",
		cqrsotel.CounterMetricWithDescription(
			"Number of SSE replays that were cut short by a timeout",
		),
	)
	if err != nil {
		return nil, err //nolint:wrapcheck // otel SDK error
	}

	return &ReplayMetrics{
		duration:        duration,
		eventsCounter:   events,
		incompleteCount: incomplete,
	}, nil
}

// RecordReplay records a completed replay: duration in milliseconds, number of
// events delivered, and whether the replay was cut short by a timeout.
// Nil-safe: a nil receiver is a no-op.
func (m *ReplayMetrics) RecordReplay(
	ctx context.Context,
	durationMs float64,
	events int,
	incomplete bool,
) {
	if m == nil {
		return
	}

	m.duration.Record(ctx, durationMs)

	if events > 0 {
		m.eventsCounter.Add(ctx, int64(events))
	}

	if incomplete {
		m.incompleteCount.Add(ctx, 1)
	}
}
