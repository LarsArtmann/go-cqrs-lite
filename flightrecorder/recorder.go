package flightrecorder

import (
	"context"
	"fmt"
	"io"
	"runtime/trace"
	"sync"
	"time"
)

const (
	defaultMinAge          = 10 * time.Second
	defaultMaxBytes uint64 = 10 << 20 // 10 MiB — ~1s of trace data for a busy service
)

// Recorder wraps [runtime/trace.FlightRecorder] with safe lifecycle
// management, configurable snapshot sinks, and once-semantics to prevent
// snapshot races.
//
// A Recorder is safe for concurrent use by multiple goroutines.
type Recorder struct {
	fr     *trace.FlightRecorder
	writer io.Writer // destination for trace snapshots

	mu   sync.Mutex // guards once, started, stopped
	once sync.Once  // ensures first Snapshot wins
}

// New creates a Recorder from the given options. Returns an error if
// the configuration is invalid.
//
// The Recorder is not started; call [Recorder.Start] to begin recording.
func New(opts ...Option) (*Recorder, error) {
	cfg := defaultConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &Recorder{
		fr: trace.NewFlightRecorder(trace.FlightRecorderConfig{
			MinAge:   cfg.minAge,
			MaxBytes: cfg.maxBytes,
		}),
		writer: cfg.writer,
	}, nil
}

// Start begins buffering execution traces in memory.
// Returns an error if the recorder is already started.
func (r *Recorder) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.fr.Start()
}

// Stop stops recording and releases the in-memory trace buffer.
// After Stop, [Recorder.Enabled] returns false and [Recorder.Snapshot]
// is a no-op. It is safe to call Stop multiple times.
func (r *Recorder) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.fr.Stop()
}

// Enabled reports whether the recorder is actively buffering traces.
func (r *Recorder) Enabled() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.fr.Enabled()
}

// Snapshot writes the buffered trace to the configured writer.
// By default, only the first successful call has effect (once-semantics)
// to prevent snapshot races when multiple goroutines detect a problem
// simultaneously. Call [Recorder.Reset] to allow subsequent captures.
//
// The snapshot is taken atomically: either the full buffer is written
// or an error is returned. If the recorder is not enabled or has already
// been snapshotted, Snapshot is a no-op and returns nil.
func (r *Recorder) Snapshot(ctx context.Context) error {
	var snapErr error

	r.once.Do(func() {
		r.mu.Lock()
		defer r.mu.Unlock()

		if !r.fr.Enabled() {
			return
		}

		if r.writer == nil {
			return
		}

		_, snapErr = r.fr.WriteTo(r.writer)
	})

	return snapErr
}

// SnapshotToFile is a convenience that writes the trace to a file.
// It creates the file, writes the snapshot, and closes the file.
// Once-semantics apply as with [Recorder.Snapshot].
func (r *Recorder) SnapshotToFile(path string) error {
	var err error

	r.once.Do(func() {
		r.mu.Lock()
		defer r.mu.Unlock()

		if !r.fr.Enabled() {
			return
		}

		f, openErr := openFile(path)
		if openErr != nil {
			err = fmt.Errorf("flightrecorder: opening snapshot file %s: %w", path, openErr)

			return
		}
		defer func() { _ = f.Close() }()

		if _, writeErr := r.fr.WriteTo(f); writeErr != nil {
			err = fmt.Errorf("flightrecorder: writing snapshot to %s: %w", path, writeErr)
		}
	})

	return err
}

// SnapshotIf evaluates the trigger against the given context and captures
// a snapshot if the trigger returns true. Returns true if a snapshot was
// initiated, false otherwise.
//
// This is the primary method for middleware integration: the middleware
// constructs a [TriggerContext] from the operation result and delegates
// the decision to the trigger function.
func (r *Recorder) SnapshotIf(ctx context.Context, tc TriggerContext, trigger TriggerFunc) bool {
	if trigger == nil || !trigger(tc) {
		return false
	}

	if err := r.Snapshot(ctx); err != nil {
		return false
	}

	return true
}

// Reset clears the once-latch so that [Recorder.Snapshot] can fire again.
// Use this when you want to capture multiple snapshots over the recorder's
// lifetime (e.g., periodic slow-operation captures).
//
// Reset does not restart a stopped recorder. Call [Recorder.Start] first
// if the recorder has been stopped.
func (r *Recorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.once = sync.Once{}
}

// Writer returns the configured snapshot writer, or nil if none was set.
func (r *Recorder) Writer() io.Writer {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.writer
}
