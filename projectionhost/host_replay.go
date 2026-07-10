package projectionhost

import (
	"context"
	"maps"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/projection/v3"
)

// host, and blocks until ctx is cancelled or all workers stop. Useful for
// simple single-projection setups.
func RegisterAndWait(ctx context.Context, h *Host, projections ...projection.Projection) error {
	for _, p := range projections {
		if err := h.Register(p); err != nil {
			return err
		}
	}

	return h.Start(ctx)
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

	result := ReplayResult{}

	for _, entry := range entries {
		w, ok := workers[entry.ProjectionName]
		if !ok {
			continue
		}

		if entry.Event == nil {
			continue
		}

		if err := w.projection.Handle(ctx, entry.Event); err != nil {
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
