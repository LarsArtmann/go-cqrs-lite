package commandlifecycle

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// Recorder writes command lifecycle events to an [event.Store]. Each method
// creates a lifecycle event and appends it to the command's lifecycle stream
// (CommandLifecycle/<cmd-id>).
//
// Versions are derived from the store, not an ephemeral counter: on first
// access to a stream the Recorder loads its current length and seeds the
// in-memory counter. This makes the Recorder safe across process restarts.
// Writes use [event.EventSink.Save] with optimistic concurrency so concurrent
// writers to the same lifecycle stream are detected instead of silently
// corrupting the stream.
//
// Recording is best-effort by default: if the sink write fails, the error is
// logged and nil is returned (the command dispatch is not affected). Use
// [WithStrict] to make recording failures propagate to the caller.
type Recorder struct {
	store  event.Store
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

// NewRecorder creates a Recorder that appends lifecycle events to store.
// The store must implement both [event.EventSink] and [event.EventSource]
// (i.e. [event.Store]) so the Recorder can derive stream versions from the
// existing event log and survive process restarts.
func NewRecorder(store event.Store, opts ...RecorderOption) *Recorder {
	r := &Recorder{
		store:    store,
		logger:   slog.Default(),
		strict:   false,
		clock:    time.Now,
		mu:       sync.Mutex{},
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
		CommandID:       CommandKey(cmd.ID().String()),
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
		CommandID:   CommandKey(cmd.ID().String()),
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

	version, err := r.nextVersion(ctx, ref.StreamKey(), ref)
	if err != nil {
		return r.handleError(err, "resolve lifecycle version", eventType, cmd)
	}

	evt, err := event.New(
		eventType,
		ref.ID,
		StreamTypeCommandLifecycle,
		version,
		payload,
		event.WithCausation(cmd.Type().String(), cmd.ID()),
		event.FromContext(ctx),
		commandTracing(cmd),
	)
	if err != nil {
		return r.handleError(err, "create lifecycle event", eventType, cmd)
	}

	if err := r.store.Save(ctx, ref, []event.Event{evt}, version-1); err != nil {
		return r.handleError(err, "append lifecycle event", eventType, cmd)
	}

	return nil
}

// metadataProvider is the optional interface a command implements to expose
// its metadata. *command.BasicCommand satisfies this.
type metadataProvider interface {
	Metadata() command.Metadata
}

// commandTracing returns an option propagating the command's tracing
// identifiers (CorrelationID, CausationID, UserID, RequestID, ActorID) onto
// the lifecycle event, so audit trails answer "who triggered the command
// that failed?". Commands that do not expose metadata are recorded without
// tracing; zero identifiers do not overwrite anything (Merge semantics).
func commandTracing(cmd command.Command) event.Option {
	mp, ok := cmd.(metadataProvider)
	if !ok {
		return func(*event.ImmutableEvent) {}
	}

	return event.WithMetadata(event.Metadata{Tracing: mp.Metadata().Tracing})
}

func (r *Recorder) nextVersion(
	ctx context.Context,
	streamKey string,
	ref id.StreamRef,
) (event.Version, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.versions[streamKey]; !ok {
		seed, err := r.seedVersion(ctx, ref)
		if err != nil {
			return 0, err
		}

		r.versions[streamKey] = seed
	}

	r.versions[streamKey]++

	return r.versions[streamKey], nil
}

func (r *Recorder) seedVersion(ctx context.Context, ref id.StreamRef) (event.Version, error) {
	existing, err := r.store.Load(ctx, ref)
	if err != nil {
		if errors.Is(err, event.ErrStreamNotFound) {
			return 0, nil
		}

		return 0, fmt.Errorf("load lifecycle stream: %w", err)
	}

	return event.Version(len(existing)), nil
}

// ResetVersion clears the cached version for a stream key, forcing the next
// emit to re-hydrate from the store. Useful when you know the stream was
// modified externally (e.g. by another process).
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
