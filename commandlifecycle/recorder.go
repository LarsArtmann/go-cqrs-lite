package commandlifecycle

import (
	"context"
	"log/slog"
	"sync"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// Recorder writes command lifecycle events to an [event.EventSink]. Each
// method creates a lifecycle event and appends it to the command's lifecycle
// stream (CommandLifecycle/<cmd-id>).
//
// Recording is best-effort by default: if the sink write fails, the error is
// logged and nil is returned (the command dispatch is not affected). Use
// [WithStrict] to make recording failures propagate to the caller.
type Recorder struct {
	sink   event.EventSink
	logger *slog.Logger
	strict bool
	clock  func() time.Time

	mu       sync.Mutex
	versions map[string]event.Version
}

// RecorderOption configures a [Recorder].
type RecorderOption func(*Recorder)

// WithLogger sets the logger for recording errors in best-effort mode.
func WithLogger(logger *slog.Logger) RecorderOption {
	return func(r *Recorder) { r.logger = logger }
}

// WithStrict makes the Recorder return errors from sink writes instead of
// logging them. Use this when lifecycle tracking is auditable and must not
// silently fail.
func WithStrict() RecorderOption {
	return func(r *Recorder) { r.strict = true }
}

// WithClock overrides the clock used for timestamps. Useful for testing.
func WithClock(clock func() time.Time) RecorderOption {
	return func(r *Recorder) { r.clock = clock }
}

// NewRecorder creates a Recorder that appends lifecycle events to sink.
func NewRecorder(sink event.EventSink, opts ...RecorderOption) *Recorder {
	r := &Recorder{
		sink:     sink,
		clock:    time.Now,
		versions: make(map[string]event.Version),
	}
	for _, opt := range opts {
		opt(r)
	}

	return r
}

// RecordReceived emits a command.received event.
func (r *Recorder) RecordReceived(ctx context.Context, cmd command.Command) error {
	return r.emit(ctx, cmd, TypeReceived, ReceivedPayload{
		CommandType:     cmd.Type().String(),
		CommandStreamID: cmd.StreamID().String(),
		ReceivedAt:      r.now(),
	})
}

// RecordFailed emits a command.failed event for a single failed attempt.
func (r *Recorder) RecordFailed(
	ctx context.Context,
	cmd command.Command,
	err error,
	attempt int,
) error {
	return r.emit(ctx, cmd, TypeFailed, FailedPayload{
		CommandType: cmd.Type().String(),
		Error:       errorMessage(err),
		Attempt:     attempt,
		FailedAt:    r.now(),
	})
}

// RecordRetried emits a command.retried event before a retry attempt.
func (r *Recorder) RecordRetried(
	ctx context.Context,
	cmd command.Command,
	attempt int,
) error {
	return r.emit(ctx, cmd, TypeRetried, RetriedPayload{
		CommandType: cmd.Type().String(),
		Attempt:     attempt,
		RetriedAt:   r.now(),
	})
}

// RecordDeadLettered emits a command.dead-lettered event after all retries
// are exhausted.
func (r *Recorder) RecordDeadLettered(
	ctx context.Context,
	cmd command.Command,
	err error,
	attempts int,
) error {
	return r.emit(ctx, cmd, TypeDeadLettered, DeadLetteredPayload{
		CommandType:    cmd.Type().String(),
		Error:          errorMessage(err),
		Attempts:       attempts,
		DeadLetteredAt: r.now(),
	})
}

// RecordCompleted emits a command.completed event after successful processing.
func (r *Recorder) RecordCompleted(ctx context.Context, cmd command.Command) error {
	return r.emit(ctx, cmd, TypeCompleted, CompletedPayload{
		CommandType: cmd.Type().String(),
		CompletedAt: r.now(),
	})
}

func (r *Recorder) emit(
	ctx context.Context,
	cmd command.Command,
	eventType event.Type,
	payload any,
) error {
	ref := LifecycleStreamRef(cmd)
	version := r.nextVersion(ref.StreamKey())

	evt, err := event.New(
		eventType,
		ref.ID,
		StreamTypeCommandLifecycle,
		version,
		payload,
		event.WithCausation(cmd.Type().String(), cmd.ID()),
		event.FromContext(ctx),
	)
	if err != nil {
		return r.handleError(err, "create lifecycle event", eventType, cmd)
	}

	if err := r.sink.AppendBatch(ctx, ref, []event.Event{evt}); err != nil {
		return r.handleError(err, "append lifecycle event", eventType, cmd)
	}

	return nil
}

func (r *Recorder) nextVersion(streamKey string) event.Version {
	r.mu.Lock()
	defer r.mu.Unlock()

	v := r.versions[streamKey] + 1
	r.versions[streamKey] = v

	return v
}

// ResetVersion resets the version counter for a stream key. This is useful
// when reconnecting to a persistent store where lifecycle events already
// exist for a command.
func (r *Recorder) ResetVersion(streamKey string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.versions, streamKey)
}

func (r *Recorder) now() time.Time {
	return r.clock()
}

func (r *Recorder) handleError(
	err error,
	operation string,
	eventType event.Type,
	cmd command.Command,
) error {
	if r.strict {
		return errorfamily.WrapInfrastructure(
			err,
			"commandlifecycle.record_failed",
			operation+" for "+string(eventType)+" on command "+cmd.ID().String(),
		)
	}

	if r.logger != nil {
		r.logger.Warn(
			"command lifecycle recording failed",
			"operation", operation,
			"eventType", string(eventType),
			"commandId", cmd.ID().String(),
			"error", err,
		)
	}

	return nil
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}
