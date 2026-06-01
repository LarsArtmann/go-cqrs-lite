package projection

import (
	"cmp"
	"context"
	"io"
	"log/slog"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/event"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
)

// Runner orchestrates projection replay from an event journal and live subscription via an event bus.
// Each registered projection tracks its own checkpoint independently.
type Runner struct {
	journal     event.Journal
	subscriber  event.Subscriber
	checkpoint  event.CheckpointStore
	opts        runnerOptions
	logger      *slog.Logger
	projections []event.Projection
	cancel      context.CancelFunc
	running     atomic.Bool
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
		return nil, event.WrapInfrastructure(ErrNilBus, "projection.create_runner",
			"create runner: nil bus")
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
	}, nil
}

// Register adds a projection to the runner. Must be called before Run.
// Returns ErrNilHandler if the projection is nil.
func (r *Runner) Register(p event.Projection) error {
	if p == nil {
		return ErrNilHandler
	}

	for _, existing := range r.projections {
		if existing.Name() == p.Name() {
			return event.WrapConflict(ErrDuplicateProjection, "projection.duplicate_name",
				"duplicate projection: "+p.Name())
		}
	}

	r.projections = append(r.projections, p)

	return nil
}

// Run replays historical events from the loader (if non-nil), then subscribes to live events.
// Blocks until the context is cancelled or Close is called. Returns ErrNoProjections if no projections are registered.
func (r *Runner) Run(ctx context.Context) error {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "projection.run",
		trace.SpanKindClient,
	)
	defer span.End()

	if len(r.projections) == 0 {
		return ErrNoProjections
	}

	r.running.Store(true)
	defer r.running.Store(false)

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

	for _, p := range r.projections {
		ctx, span := cqrsotel.StartSpan(
			ctx, tracer(), "projection.replay",
			trace.SpanKindClient,
			trace.WithAttributes(projectionAttrs(p.Name())...),
		)

		checkpoint, cpErr := r.checkpoint.Load(ctx, p.Name())
		if cpErr != nil {
			cqrsotel.RecordError(span, cpErr)
			span.End()

			return event.WrapInfrastructure(cpErr, "projection.load_checkpoint",
				"load checkpoint for "+p.Name())
		}

		var events []event.Event

		if hasSeekable && !checkpoint.IsZero() {
			loaded, lErr := seekable.ReadFrom(ctx, checkpoint.EventID, 0)
			if lErr != nil {
				cqrsotel.RecordError(span, lErr)
				span.End()

				return event.WrapInfrastructure(lErr, "projection.load_events",
					"load events from position for "+p.Name())
			}

			events = filterByEventTypes(loaded, p.EventTypes())
		} else {
			allEvents, lErr := r.journal.ReadAll(ctx)
			if lErr != nil {
				cqrsotel.RecordError(span, lErr)
				span.End()

				return event.WrapInfrastructure(lErr, "projection.load_events",
					"load all events")
			}

			events = filterFromCheckpoint(allEvents, p.EventTypes(), checkpoint)
		}

		span.SetAttributes(attribute.Int(cqrsotel.AttrEventCount, len(events)))

		for _, evt := range events {
			replayCtx := event.WithReplay(ctx, true)

			hErr := r.handleAndCheckpoint(replayCtx, p, evt)
			if hErr != nil {
				cqrsotel.RecordError(span, hErr)
				span.End()

				return event.WrapCorruption(hErr, "projection.replay_event",
					"replay "+p.Name()+" event "+evt.ID().String())
			}
		}

		span.End()
	}

	return nil
}

func (r *Runner) handleAndCheckpoint(
	ctx context.Context,
	p event.Projection,
	evt event.Event,
) error {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "projection.handle",
		trace.SpanKindConsumer,
		trace.WithAttributes(
			attribute.String(cqrsotel.AttrEventType, string(evt.Type())),
			attribute.String(cqrsotel.AttrProjectionName, p.Name()),
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

// Close cancels the internal context, causing Run to return gracefully.
func (r *Runner) Close() error {
	r.cancel()

	return nil
}
