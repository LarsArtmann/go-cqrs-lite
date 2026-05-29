package projection

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event"
)

func (r *Runner) subscribeLive(ctx context.Context) error {
	handler := func(ctx context.Context, evt event.Event) error {
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
	sem := make(chan struct{}, r.opts.parallelism)

	var wg sync.WaitGroup

	for _, p := range projections {
		wg.Add(1)

		go func(p event.Projection) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			r.dispatchOne(ctx, p, evt)
		}(p)
	}

	wg.Wait()
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

		select {
		case <-ctx.Done():
			return event.WrapInfrastructure(ctx.Err(), "projection.retry_cancelled",
				"retry cancelled")
		case <-time.After(delay):
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
