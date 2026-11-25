package metaengine

import "context"

// ApplyIdempotent processes an event with deduplication by event ID. If the
// same eventID has been applied before, the call is a no-op. This is essential
// for at-least-once delivery scenarios where events may be replayed.
//
// The dedup is in-memory (process-local); for durable dedup across restarts,
// consumers should wrap the Store with an external idempotency store.
//
// The ID is recorded only after the apply succeeded: a failed call returns the
// error and leaves the ID unrecorded, so a retry of the same event ID applies.
// A duplicate that arrives while the first apply is still running is a no-op.
func (s *Store) ApplyIdempotent(ctx context.Context, eventID, eventType string, payload any) error {
	if eventID == "" {
		return s.applyIdempotentFeed(ctx, eventType, payload)
	}

	if s.idempotency.claim(eventID) {
		return nil // already applied, or being applied by another goroutine
	}

	err := s.applyIdempotentFeed(ctx, eventType, payload)
	if err != nil {
		s.idempotency.release(eventID)

		return err
	}

	s.idempotency.commit(eventID)

	return nil
}
