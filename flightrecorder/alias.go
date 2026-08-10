package flightrecorder

import (
	"context"
	"time"

	goflightrecorder "github.com/larsartmann/go-flightrecorder"
)

// Recorder wraps runtime/trace.FlightRecorder with safe lifecycle management.
// Deprecated: use github.com/larsartmann/go-flightrecorder.Recorder.
type Recorder = goflightrecorder.Recorder

// Option configures a Recorder.
// Deprecated: use github.com/larsartmann/go-flightrecorder.Option.
type Option = goflightrecorder.Option

// TriggerContext describes an operation that just completed.
// Deprecated: use github.com/larsartmann/go-flightrecorder.TriggerContext.
type TriggerContext = goflightrecorder.TriggerContext

// TriggerFunc decides whether a flight recorder snapshot should be captured.
// Deprecated: use github.com/larsartmann/go-flightrecorder.TriggerFunc.
type TriggerFunc = goflightrecorder.TriggerFunc

// ErrAlreadyEnabled is returned when another flight recorder is already active.
// Deprecated: use github.com/larsartmann/go-flightrecorder.ErrAlreadyEnabled.
var ErrAlreadyEnabled = goflightrecorder.ErrAlreadyEnabled

// New creates a Recorder from the given options.
// Deprecated: use github.com/larsartmann/go-flightrecorder.New.
func New(opts ...Option) (*Recorder, error) {
	return goflightrecorder.New(opts...)
}

// WithMinAge sets the minimum age of trace data that is reliably retained.
// Deprecated: use github.com/larsartmann/go-flightrecorder.WithMinAge.
func WithMinAge(d time.Duration) Option {
	return goflightrecorder.WithMinAge(d)
}

// WithMaxBytes sets the maximum size of the in-memory trace buffer.
// Deprecated: use github.com/larsartmann/go-flightrecorder.WithMaxBytes.
func WithMaxBytes(n uint64) Option {
	return goflightrecorder.WithMaxBytes(n)
}

// WithWriter sets the destination for Snapshot writes.
// Deprecated: use github.com/larsartmann/go-flightrecorder.WithWriter.
func WithWriter(w interface{ Write([]byte) (int, error) }) Option {
	return goflightrecorder.WithWriter(w)
}

// WithFile sets the snapshot destination to a file at the given path.
// Deprecated: use github.com/larsartmann/go-flightrecorder.WithFile.
func WithFile(path string) Option {
	return goflightrecorder.WithFile(path)
}

// OnLatency returns a trigger that fires when duration exceeds threshold.
// Deprecated: use github.com/larsartmann/go-flightrecorder.OnLatency.
func OnLatency(threshold time.Duration) TriggerFunc {
	return goflightrecorder.OnLatency(threshold)
}

// OnError returns a trigger that fires when an operation returns a non-nil error.
// Deprecated: use github.com/larsartmann/go-flightrecorder.OnError.
func OnError() TriggerFunc {
	return goflightrecorder.OnError()
}

// OnErrorOrLatency returns a trigger that fires on error OR latency exceeding threshold.
// Deprecated: use github.com/larsartmann/go-flightrecorder.OnErrorOrLatency.
func OnErrorOrLatency(threshold time.Duration) TriggerFunc {
	return goflightrecorder.OnErrorOrLatency(threshold)
}

// OnAlways returns a trigger that always fires.
// Deprecated: use github.com/larsartmann/go-flightrecorder.OnAlways.
func OnAlways() TriggerFunc {
	return goflightrecorder.OnAlways()
}

// OnAny returns a trigger that fires if any of the given triggers fire.
// Deprecated: use github.com/larsartmann/go-flightrecorder.OnAny.
func OnAny(triggers ...TriggerFunc) TriggerFunc {
	return goflightrecorder.OnAny(triggers...)
}

// OnAll returns a trigger that fires only if all given triggers fire.
// Deprecated: use github.com/larsartmann/go-flightrecorder.OnAll.
func OnAll(triggers ...TriggerFunc) TriggerFunc {
	return goflightrecorder.OnAll(triggers...)
}

// SnapshotToFile is a convenience that writes the trace to a file.
// Deprecated: use github.com/larsartmann/go-flightrecorder.Recorder.SnapshotToFile.
//
//nolint:wrapcheck // pure re-export alias
func SnapshotToFile(r *Recorder, ctx context.Context, path string) error {
	return r.SnapshotToFile(ctx, path)
}

// SnapshotIf evaluates the trigger and captures a snapshot if it returns true.
// Deprecated: use github.com/larsartmann/go-flightrecorder.Recorder.SnapshotIf.
//

func SnapshotIf(r *Recorder, ctx context.Context, tc TriggerContext, trigger TriggerFunc) bool {
	return r.SnapshotIf(ctx, tc, trigger)
}
