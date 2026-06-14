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
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
)

// projectionEntry pairs a Projection with its event types cached at registration time.
// This eliminates per-event EventTypes() calls (which clone) on the hot dispatch path.
type projectionEntry struct {
	projection event.Projection
	eventTypes []event.Type
}

// Runner orchestrates projection replay from an event journal and live subscription via an event bus.
// Each registered projection tracks its own checkpoint independently.
type Runner struct {
	journal     event.Journal
	subscriber  event.Subscriber
	checkpoint  event.CheckpointStore
	opts        runnerOptions
	logger      *slog.Logger
	projections []projectionEntry
	cancel      context.CancelFunc
	running     atomic.Bool
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

// Run replays historical events from the loader (if non-nil), then subscribes to live events.
// Blocks until the context is cancelled or Close is called. Returns ErrNoProjections if no projections are registered.
func (r *Runner) Run(ctx context.Context) error {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "projection.run",
		cqrsotel.SpanKindClient,
	)
	defer span.End()

	if len(r.projections) == 0 {
		return ErrNoProjections
	}

	if !r.running.CompareAndSwap(false, true) {
		return ErrAlreadyRunning
	}
	defer r.running.Store(false)

	done := make(chan struct{})

	r.done = done
	defer close(done)

	ctx, r.cancel = context.WithCancel(ctx)

	if r.journal != nil {
		err := r.replay(ctx)
		if err != nil {
			return event.WrapInfrastructure(err, "projection.replay",
				"replay failed")
		}
	}

	return r.subscribeLive(ctx)
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

// Close cancels the internal context and waits for Run to return.
// Safe to call multiple times. If Run was never called, returns immediately.
func (r *Runner) Close() error {
	r.closeOnce.Do(func() {
		r.cancel()
	})

	if r.running.Load() {
		<-r.done
	}

	return nil
}
