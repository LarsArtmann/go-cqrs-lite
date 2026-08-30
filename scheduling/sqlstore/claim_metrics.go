package sqlstore

// ClaimMetrics is the opt-in, zero-dependency observability surface for
// [ClaimingTimerStore]. scheduling deliberately carries no OpenTelemetry
// dependency (lean-budget module), so callers who want counters wire their
// own — e.g. incrementing an otel.Int64Counter inside each hook:
//
//	metrics := sqlstore.ClaimMetrics{
//	    Claimed:       func(n int) { claimed.Add(ctx, int64(n)) },
//	    Renewed:       func() { renewed.Add(ctx, 1) },
//	    RenewRejected: func() { rejected.Add(ctx, 1) },
//	}
//	store, err := sqlstore.NewClaimingPostgresStore[Payload](ctx, db, lease,
//	    sqlstore.WithClaimMetrics[Payload](metrics))
//
// Hooks run synchronously in the polling/dispatch goroutine while the store
// holds no lock; they must be cheap and must not call back into the store.
type ClaimMetrics struct {
	// Claimed is called after a Due claim commits, with the number of timers
	// claimed in that batch (0 when the poll found nothing due).
	Claimed func(count int)

	// Renewed is called after RenewLease extends a live lease.
	Renewed func()

	// RenewRejected is called when RenewLease fails with ErrLeaseNotHeld —
	// the lease expired, the timer fired, was cancelled, or does not exist.
	RenewRejected func()
}

// ClaimOption configures a [ClaimingTimerStore] at construction.
type ClaimOption[P any] func(*ClaimingTimerStore[P])

// WithClaimMetrics attaches opt-in observability hooks to the claiming store.
// Passing ClaimMetrics{} (or calling WithClaimMetrics with all-nil hooks)
// leaves the store unobserved.
func WithClaimMetrics[P any](m ClaimMetrics) ClaimOption[P] {
	return func(c *ClaimingTimerStore[P]) { c.metrics = m }
}
