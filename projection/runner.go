package projection

import (
	"cmp"
	"context"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
)

// projectionEntry pairs a Projection with its event types cached at registration time.
// This eliminates per-event EventTypes() calls (which clone) on the hot dispatch path.
type projectionEntry struct {
	projection event.Projection
	eventTypes []event.Type
}

// runnerState* are the Runner lifecycle states, guarded by Runner.state.
// The runner transitions idle → ready (RunReplay) → live (RunLive) → idle.
const (
	runnerStateIdle  int32 = 0 // no run active; RunReplay is allowed
	runnerStateReady int32 = 1 // RunReplay completed; RunLive is allowed
	runnerStateLive  int32 = 2 // RunLive is blocking on the live subscription
)

// Runner orchestrates projection replay from an event journal and live subscription via an event bus.
// Each registered projection tracks its own checkpoint independently.
type Runner struct {
	journal     event.Journal
	subscriber  event.Subscriber
	checkpoint  event.CheckpointStore
	opts        runnerOptions
	logger      *slog.Logger
	projections []projectionEntry
	replayIDs   map[id.EventID]struct{}
	cancel      context.CancelFunc
	state       atomic.Int32
	done        chan struct{}
	closeOnce   sync.Once
}

var _ io.Closer = (*Runner)(nil)

// NewRunner creates a projection Runner. Pass a nil journal to skip replay (live-only mode).
// Returns an error if subscriber or checkpoint is nil.
func NewRunner(
	journal event.Journal,
	subscriber event.Subscriber,
	checkpoint event.CheckpointStore,
	opts ...RunnerOption,
) (*Runner, error) {
	if subscriber == nil {
		return nil, event.WrapInfrastructure(ErrNilSubscriber, "projection.create_runner",
			"create runner: nil subscriber")
	}

	if checkpoint == nil {
		return nil, event.WrapInfrastructure(ErrNilCheckpoint, "projection.create_runner",
			"create runner: nil checkpoint")
	}

	o := runnerOptions{}

	for _, opt := range opts {
		opt(&o)
	}

	logger := cmp.Or(o.logger, slog.Default())

	cancel := context.CancelFunc(func() {})

	return &Runner{
		journal:    journal,
		subscriber: subscriber,
		checkpoint: checkpoint,
		opts:       o,
		logger:     logger,
		replayIDs:  make(map[id.EventID]struct{}),
		cancel:     cancel,
		done:       make(chan struct{}),
	}, nil
}

// Register adds a projection to the runner. Must be called before Run.
// Returns ErrNilHandler if the projection is nil.
func (r *Runner) Register(p event.Projection) error {
	if p == nil {
		return ErrNilHandler
	}

	for _, existing := range r.projections {
		if existing.projection.Name() == p.Name() {
			return event.WrapConflict(ErrDuplicateProjection, "projection.duplicate_name",
				"duplicate projection: "+p.Name())
		}
	}

	r.projections = append(r.projections, projectionEntry{
		projection: p,
		eventTypes: p.EventTypes(),
	})

	return nil
}

// RunReplay replays historical events from the journal (if non-nil) and returns
// once every registered projection has caught up to the current event stream.
// It is the synchronous, non-blocking half of the projection lifecycle and
// enables read-your-writes consistency: after RunReplay returns, the read model
// reflects all previously committed events, so callers may query it immediately.
//
// RunReplay must be followed by RunLive (or use Run, which calls both). It
// returns ErrNoProjections if no projections are registered, ErrAlreadyRunning
// if the runner is active, and a wrapped Infrastructure error on replay failure
// (the runner returns to idle and may be retried).
func (r *Runner) RunReplay(ctx context.Context) error {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "projection.run_replay",
		cqrsotel.SpanKindClient,
	)
	defer span.End()

	if len(r.projections) == 0 {
		return ErrNoProjections
	}

	if !r.state.CompareAndSwap(runnerStateIdle, runnerStateReady) {
		return ErrAlreadyRunning
	}

	if r.journal == nil {
		return nil
	}

	r.replayIDs = make(map[id.EventID]struct{})

	err := r.replay(ctx)
	if err != nil {
		r.state.Store(runnerStateIdle)

		return event.WrapInfrastructure(err, "projection.replay", "replay failed")
	}

	return nil
}

// RunLive subscribes to live events from the bus and blocks until ctx is
// cancelled or Close is called. RunReplay must have completed successfully
// first; otherwise RunLive returns ErrReplayRequired.
func (r *Runner) RunLive(ctx context.Context) error {
	if r.state.Load() == runnerStateIdle {
		return ErrReplayRequired
	}

	if r.state.Load() == runnerStateLive {
		return ErrAlreadyRunning
	}

	if !r.state.CompareAndSwap(runnerStateReady, runnerStateLive) {
		return ErrAlreadyRunning
	}

	// We own the live phase: assign shutdown fields AFTER winning the CAS so a
	// concurrent (losing) RunLive caller cannot clobber them.
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(ctx)

	r.done = done
	r.cancel = cancel

	defer cancel()

	defer func() {
		r.state.Store(runnerStateIdle)
		close(done)
	}()

	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "projection.run_live",
		cqrsotel.SpanKindClient,
	)
	defer span.End()

	return r.subscribeLive(ctx)
}

// Run is a convenience wrapper that calls RunReplay followed by RunLive.
// It replays historical events from the journal (if non-nil), then subscribes to
// live events. Blocks until the context is cancelled or Close is called.
// Returns ErrNoProjections if no projections are registered.
func (r *Runner) Run(ctx context.Context) error {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "projection.run",
		cqrsotel.SpanKindClient,
	)
	defer span.End()

	err := r.RunReplay(ctx)
	if err != nil {
		return err
	}

	return r.RunLive(ctx)
}

func (r *Runner) replay(ctx context.Context) error {
	seekable, hasSeekable := r.journal.(event.SeekableJournal)

	for _, entry := range r.projections {
		ctx, span := cqrsotel.StartSpan(
			ctx, tracer(), "projection.replay",
			cqrsotel.SpanKindClient,
			cqrsotel.WithAttributes(projectionAttrs(entry.projection.Name())...),
		)

		events, err := r.loadReplayEvents(
			ctx,
			seekable,
			hasSeekable,
			entry.projection,
			entry.eventTypes,
		)
		if err != nil {
			cqrsotel.RecordError(span, err)
			span.End()

			return err
		}

		span.SetAttributes(cqrsotel.AttrInt(cqrsotel.AttrEventCount, len(events)))

		for _, evt := range events {
			r.replayIDs[evt.ID()] = struct{}{}

			replayCtx := event.WithProcessingMode(ctx, event.ModeReplay)

			hErr := r.handleAndCheckpoint(replayCtx, entry.projection, evt)
			if hErr != nil {
				cqrsotel.RecordError(span, hErr)
				span.End()

				return event.WrapCorruption(hErr, "projection.replay_event",
					"replay "+entry.projection.Name()+" event "+evt.ID().String())
			}
		}

		span.End()
	}

	return nil
}

func (r *Runner) loadReplayEvents(
	ctx context.Context,
	seekable event.SeekableJournal,
	hasSeekable bool,
	p event.Projection,
	eventTypes []event.Type,
) ([]event.Event, error) {
	checkpoint, cpErr := r.checkpoint.Load(ctx, p.Name())
	if cpErr != nil {
		return nil, event.WrapInfrastructure(cpErr, "projection.load_checkpoint",
			"load checkpoint for "+p.Name())
	}

	if hasSeekable && !checkpoint.IsZero() {
		loaded, lErr := seekable.ReadFrom(ctx, checkpoint.EventID, 0)
		if lErr != nil {
			return nil, event.WrapInfrastructure(lErr, "projection.load_events",
				"load events from position for "+p.Name())
		}

		return filterByEventTypes(loaded, eventTypes), nil
	}

	allEvents, lErr := r.journal.ReadAll(ctx)
	if lErr != nil {
		return nil, event.WrapInfrastructure(lErr, "projection.load_events",
			"load all events")
	}

	return filterFromCheckpoint(allEvents, eventTypes, checkpoint), nil
}

func (r *Runner) handleAndCheckpoint(
	ctx context.Context,
	p event.Projection,
	evt event.Event,
) error {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "projection.handle",
		cqrsotel.SpanKindConsumer,
		cqrsotel.WithAttributes(
			cqrsotel.AttrString(cqrsotel.AttrEventType, string(evt.Type())),
			cqrsotel.AttrString(cqrsotel.AttrProjectionName, p.Name()),
		),
	)
	defer span.End()

	err := p.Handle(ctx, evt)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return event.Wrap(err, event.Classify(err), "projection.handle_event",
			"projection "+p.Name()+" handle event "+string(evt.Type()))
	}

	cqrsotel.AddSpanEvent(
		span, "checkpoint_saved",
		cqrsotel.AttrString("projection", p.Name()),
		cqrsotel.AttrString("event_id", evt.ID().String()),
	)

	return r.checkpoint.Save(
		ctx,
		p.Name(),
		event.Checkpoint{EventID: evt.ID(), ProcessedAt: time.Now()},
	)
}

// CurrentCheckpoint returns the last processed event ID for the given projection.
func (r *Runner) CurrentCheckpoint(
	ctx context.Context,
	projectionName string,
) (event.Checkpoint, error) {
	return r.checkpoint.Load(ctx, projectionName)
}

// Reset clears the checkpoint for a projection, allowing full replay on the next Run.
func (r *Runner) Reset(ctx context.Context, projectionName string) error {
	return r.checkpoint.Save(ctx, projectionName, event.Checkpoint{})
}

// Close cancels the run context and waits for RunLive to return.
// Safe to call multiple times. If the runner never reached the live phase it
// returns immediately. A ready-only lifecycle (RunReplay without RunLive) is
// reset to idle so the runner can be reused.
func (r *Runner) Close() error {
	r.closeOnce.Do(func() {
		r.cancel()
	})

	if r.state.Load() == runnerStateLive {
		<-r.done
	}

	r.state.CompareAndSwap(runnerStateReady, runnerStateIdle)

	return nil
}
