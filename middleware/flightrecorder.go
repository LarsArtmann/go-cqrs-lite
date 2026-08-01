package middleware

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	flightrecorder "github.com/larsartmann/go-cqrs-lite/flightrecorder/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// NewFlightRecorder returns a generic middleware that captures a flight
// recorder snapshot when the trigger condition is met. The snapshot is
// captured asynchronously to avoid blocking the request path.
//
// If recorder is nil, the middleware is a pass-through.
// If trigger is nil, no snapshots are captured.
func NewFlightRecorder[M any](
	adapter MessageAdapter[M],
	recorder *flightrecorder.Recorder,
	trigger flightrecorder.TriggerFunc,
	opts ...Option,
) Middleware[M] {
	cfg := applyOptions(opts)

	return func(next Handler[M]) Handler[M] {
		return func(ctx context.Context, msg M) (err error) {
			start := time.Now()

			err = next(ctx, msg)

			if recorder == nil || trigger == nil {
				return err
			}

			tc := flightrecorder.TriggerContext{
				Kind:     adapter.Kind,
				Type:     adapter.ExtractType(msg),
				Duration: time.Since(start),
				Err:      err,
			}

			if !trigger(tc) {
				return err
			}

			go func() {
				if snapErr := recorder.Snapshot(ctx); snapErr != nil && cfg.logger != nil {
					cfg.logger.Error(
						"flight recorder snapshot failed",
						"kind", adapter.Kind,
						"type", tc.Type,
						"duration", tc.Duration,
						"error", snapErr,
					)
				}
			}()

			return err
		}
	}
}

// CommandFlightRecorder returns a command middleware that captures a flight
// recorder snapshot when the trigger fires.
//
//	recorder, _ := flightrecorder.New(
//	    flightrecorder.WithMinAge(10*time.Second),
//	    flightrecorder.WithFile("snapshot.trace"),
//	)
//	recorder.Start()
//	defer recorder.Stop()
//
//	cmdDisp.Use(middleware.CommandFlightRecorder(recorder,
//	    flightrecorder.OnErrorOrLatency(100*time.Millisecond)))
func CommandFlightRecorder(
	recorder *flightrecorder.Recorder,
	trigger flightrecorder.TriggerFunc,
	opts ...Option,
) command.Middleware {
	return AsCommand(NewFlightRecorder(CommandAdapter, recorder, trigger, opts...))
}

// EventFlightRecorder returns an event middleware that captures a flight
// recorder snapshot when the trigger fires.
func EventFlightRecorder(
	recorder *flightrecorder.Recorder,
	trigger flightrecorder.TriggerFunc,
	opts ...Option,
) event.Middleware {
	return AsEvent(NewFlightRecorder(EventAdapter, recorder, trigger, opts...))
}

// QueryFlightRecorder returns a query middleware that captures a flight
// recorder snapshot when the trigger fires.
func QueryFlightRecorder(
	recorder *flightrecorder.Recorder,
	trigger flightrecorder.TriggerFunc,
	opts ...Option,
) query.Middleware {
	return AsQuery(NewFlightRecorder(QueryAdapter, recorder, trigger, opts...))
}
