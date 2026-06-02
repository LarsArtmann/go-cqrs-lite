package projection

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"golang.org/x/sync/errgroup"
)

func (r *Runner) subscribeLive(ctx context.Context) error {
	handler := func(ctx context.Context, evt event.Event) error {
		if !r.running.Load() {
			return nil
		}

		r.dispatchToProjections(ctx, evt)

		return nil
	}

	err := r.subscriber.SubscribeAll(handler)
	if err != nil {
		return event.WrapInfrastructure(err, "projection.subscribe",
			"subscribe to event bus")
	}

	<-ctx.Done()

	return nil
}

func (r *Runner) dispatchToProjections(ctx context.Context, evt event.Event) {
	var candidates []event.Projection

	for _, p := range r.projections {
		if subscribesTo(p, evt.Type()) {
			candidates = append(candidates, p)
		}
	}

	if len(candidates) == 0 {
		return
	}

	if r.opts.parallelism > 1 {
		r.dispatchParallel(ctx, evt, candidates)

		return
	}

	for _, p := range candidates {
		r.dispatchOne(ctx, p, evt)
	}
}

func (r *Runner) dispatchParallel(
	ctx context.Context,
	evt event.Event,
	projections []event.Projection,
) {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(r.opts.parallelism)

	for _, p := range projections {
		g.Go(func() error {
			r.dispatchOne(ctx, p, evt)

			return nil
		})
	}

	_ = g.Wait()
}

func (r *Runner) dispatchOne(ctx context.Context, p event.Projection, evt event.Event) {
	err := r.handleWithRetry(ctx, p, evt)
	if err != nil {
		r.logger.ErrorContext(
			ctx, "projection handler failed",
			"projection", p.Name(),
			"event_id", evt.ID(),
			"event_type", evt.Type(),
			"error", err,
		)

		if r.opts.deadLetter != nil {
			r.opts.deadLetter(ctx, p.Name(), evt, err)
		}
	}
}

func (r *Runner) handleWithRetry(ctx context.Context, p event.Projection, evt event.Event) error {
	err := r.handleAndCheckpoint(ctx, p, evt)
	if err == nil {
		return nil
	}

	if r.opts.retryCount <= 0 || !event.IsRetryable(err) {
		return event.WrapCorruption(err, "projection.non_retryable",
			"projection "+p.Name()+" non-retryable error")
	}

	for attempt := 1; attempt <= r.opts.retryCount; attempt++ {
		delay := r.opts.retryDelay * time.Duration(1<<(attempt-1))
		if r.opts.retryMaxDelay > 0 && delay > r.opts.retryMaxDelay {
			delay = r.opts.retryMaxDelay
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()

			return event.WrapInfrastructure(ctx.Err(), "projection.retry_cancelled",
				"retry cancelled")
		case <-timer.C:
		}

		err = r.handleAndCheckpoint(ctx, p, evt)
		if err == nil {
			return nil
		}
	}

	return event.WrapTransient(err, "projection.retry_exhausted",
		fmt.Sprintf("projection %q retry exhausted after %d attempts",
			p.Name(), r.opts.retryCount))
}
