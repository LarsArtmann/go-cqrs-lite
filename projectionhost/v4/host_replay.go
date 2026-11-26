package projectionhost

import (
	"context"
	"maps"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/projection/v4"
)

// RegisterAndWait registers all projections with the host, starts the host,
// and blocks until ctx is cancelled or all workers stop. Useful for simple
// single-projection setups; call it from the goroutine that owns shutdown.
//
// Cancelling ctx stops the host gracefully (as [Host.Stop]), so ctx should
// live as long as the process. It returns nil after a clean stop and the
// error if a projection could not be registered or the host could not start
// or stop.
func RegisterAndWait(ctx context.Context, h *Host, projections ...projection.Projection) error {
	for _, p := range projections {
		if err := h.Register(p); err != nil {
			return err
		}
	}

	// The workers must not inherit ctx's cancellation: stopping is routed
	// through Host.Stop below, so a cancelled ctx reads as a shutdown and
	// not as a Start context that died underneath a running host.
	if err := h.Start(context.WithoutCancel(ctx)); err != nil {
		return err
	}

	return h.waitUntilStopped(ctx)
}

// waitUntilStopped blocks until ctx is cancelled and the host is stopped, or
// until every worker has exited on its own.
func (h *Host) waitUntilStopped(ctx context.Context) error {
	exited := make(chan struct{})

	go func() {
		h.wg.Wait()
		close(exited)
	}()

	select {
	case <-ctx.Done():
		//nolint:contextcheck // Stop's frozen v4 signature takes no ctx; it bounds shutdown by its own shutdownTimeout
		return h.Stop()
	case <-exited:
		return nil
	}
}

// ReplayResult reports the outcome of a pure ReplayDeadLetters run.
// ReplayDeadLetters does NOT mutate the DeadLetterStore — callers decide
// whether to Purge the successfully replayed entries.
type ReplayResult struct {
	// Replayed are the entries whose handler now succeeds.
	Replayed []DeadLetterEntry
	// StillFailing are the entries whose handler still returns an error.
	StillFailing []ReplayFailure
}

// ReplayFailure pairs a dead-letter entry with the error its handler returned
// during the replay attempt.
type ReplayFailure struct {
	Entry DeadLetterEntry
	Err   error
}

// ReplayDeadLetters re-feeds dead-letter entries to the matching registered
// projection WITHOUT mutating the store. Pure retry: the caller decides whether
// to call Purge afterwards. Use this after fixing the handler bug that
// originally poisoned the events.
//
// An empty projectionName replays across all projections. The Host must have a
// DeadLetterStore configured (WithDeadLetterStore); otherwise ReplayDeadLetters
// returns an error. The Host need not be running.
//
// Entries whose projection is not registered, or whose Event field is nil, are
// skipped silently (counted in neither Replayed nor StillFailing).
func (h *Host) ReplayDeadLetters(ctx context.Context, projectionName string) (ReplayResult, error) {
	h.mu.Lock()
	dlq := h.opts.dlq
	workers := maps.Clone(h.workers)
	h.mu.Unlock()

	if dlq == nil {
		return ReplayResult{}, errorfamily.NewRejection(
			"projectionhost.no_dead_letter_store",
			"projectionhost: no dead-letter store configured",
		)
	}

	entries, err := dlq.List(ctx, projectionName)
	if err != nil {
		return ReplayResult{}, errorfamily.WrapInfrastructure(
			err,
			"projectionhost.list_dead_letters",
			"list dead letters for replay",
		)
	}

	result := ReplayResult{} //nolint:exhaustruct_v5 // zero-value init

	for _, entry := range entries {
		w, ok := workers[entry.ProjectionName]
		if !ok {
			continue
		}

		if entry.Event == nil {
			continue
		}

		// handleMu serializes with a concurrently running worker's drain
		// and live callback — the same lock the worker itself takes.
		w.handleMu.Lock()
		err := w.projection.Handle(ctx, entry.Event)
		w.handleMu.Unlock()

		if err != nil {
			h.opts.logger.Warn("dead-letter replay still failing",
				"projection", entry.ProjectionName,
				"event_id", entry.EventID, "error", err)
			result.StillFailing = append(result.StillFailing, ReplayFailure{Entry: entry, Err: err})

			continue
		}

		result.Replayed = append(result.Replayed, entry)
	}

	return result, nil
}
