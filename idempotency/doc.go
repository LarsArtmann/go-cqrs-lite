// Package idempotency provides a deduplication store for command idempotency
// keys (and any other opaque at-most-once-processing keys).
//
// Delivery in a CQRS system is at-least-once: a client may submit a command,
// lose the acknowledgement, and retry. Without deduplication, the retried
// command executes twice and produces duplicate events and side effects.
//
// An idempotency store closes that gap. The client attaches a stable key to
// each logical command; the server records the key before processing and
// rejects retries whose key has already been recorded.
//
// # Quick Start
//
//	store := idempotency.NewMemoryStore(5 * time.Minute)
//	defer store.Close()
//
//	// Check-and-record in a single atomic step (preferred over Seen + Record).
//	err := store.CheckAndRecord(ctx, cmdKey, 10*time.Minute)
//	if errors.Is(err, idempotency.ErrDuplicate) {
//	    return err // already processed — drop the retry
//	}
//	if err != nil {
//	    return err // store failure — do not process
//	}
//
// This module owns the storage primitive only. For middleware that wires the
// store into command, event, or query dispatch pipelines, import the
// middleware package and use CommandIdempotency, EventIdempotency, or
// QueryIdempotency. For custom integrations (transport hooks, manual
// checks), use the Store interface directly.
//
// The canonical implementation lives in [github.com/larsartmann/go-idempotency];
// this package re-exports it for backward compatibility (ADR-0065).
package idempotency
