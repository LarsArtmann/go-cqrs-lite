package decider

import (
	"context"
	"log/slog"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	flightrecorder "github.com/larsartmann/go-flightrecorder"
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

	// The snapshot runs after Execute returns, so the request context may
	// already be cancelled (especially on error paths where the client gave
	// up). Detach cancellation but keep tracing values.
	snapshotCtx := context.WithoutCancel(ctx)

	go func() {
		if snapErr := r.flightRecorder.Snapshot(snapshotCtx); snapErr != nil {
			slog.WarnContext(snapshotCtx, "flight recorder snapshot failed in decider.Execute",
				"ref", ref.String(), "error", snapErr)
		}
	}()
}
