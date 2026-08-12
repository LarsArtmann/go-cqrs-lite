package decider

import (
	"context"
	"log/slog"
	"time"

	flightrecorder "github.com/larsartmann/go-cqrs-lite/flightrecorder/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// maybeCaptureFlightRecorder evaluates the flight recorder trigger after
// Execute completes. If the trigger fires, a snapshot is captured
// asynchronously (goroutine) to avoid blocking the request path.
// No-op when no flight recorder is configured.
func (r *Repository[State]) maybeCaptureFlightRecorder(
	ctx context.Context,
	ref id.StreamRef,
	streamType id.StreamType,
	execErr error,
	duration time.Duration,
) {
	if r.flightRecorder == nil {
		return
	}

	trigger := r.flightRecorderTrigger
	if trigger == nil {
		trigger = flightrecorder.OnError()
	}

	tc := flightrecorder.TriggerContext{
		Kind:     "decider",
		Type:     string(streamType),
		Duration: duration,
		Err:      execErr,
	}

	if !trigger(tc) {
		return
	}

	go func() {
		if snapErr := r.flightRecorder.Snapshot(ctx); snapErr != nil {
			slog.WarnContext(ctx, "flight recorder snapshot failed in decider.Execute",
				"ref", ref.String(), "error", snapErr)
		}
	}()
}
