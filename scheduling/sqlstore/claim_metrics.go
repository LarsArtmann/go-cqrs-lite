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
// For the common "just show the numbers" case the store ALSO maintains
// built-in counters itself — no hooks required: read them via
// [ClaimingTimerStore.Metrics] from a Doctor-style report or a status
// endpoint.
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
// leaves the user hooks unwired; the built-in counters behind Metrics are
// maintained regardless.
func WithClaimMetrics[P any](m ClaimMetrics) ClaimOption[P] {
	return func(c *ClaimingTimerStore[P]) { c.metrics = m }
}

// ClaimMetricsSnapshot is the store-maintained view of claim activity: the
// numbers a status endpoint or health report shows without any hook wiring.
// Field semantics mirror the ClaimMetrics hooks; ClaimedBatches counts EVERY
// committed Due poll, including polls that claimed nothing — an empty batch
// is the liveness heartbeat of a poller.
//
// Counters are process-local (they reset on restart, like the hooks); they
// observe THIS store instance only — a second poller on the same timers table
// keeps its own counts.
type ClaimMetricsSnapshot struct {
	// ClaimedBatches counts committed Due polls, including empty ones.
	ClaimedBatches int64 `json:"claimed_batches"`

	// ClaimedTimers counts timers claimed across all polls.
	ClaimedTimers int64 `json:"claimed_timers"`

	// Renewed counts successful RenewLease extensions.
	Renewed int64 `json:"renewed"`

	// RenewRejected counts renewals rejected with ErrLeaseNotHeld.
	RenewRejected int64 `json:"renew_rejected"`
}

// Metrics returns a snapshot of the built-in claim counters. It is always
// available — independent of WithClaimMetrics — so operators can surface
// claim activity in Doctor-style diagnostics or a /status endpoint without
// wiring hooks. Snapshot fields are plain values: safe to marshal, safe to
// copy.
func (c *ClaimingTimerStore[P]) Metrics() ClaimMetricsSnapshot {
	return ClaimMetricsSnapshot{
		ClaimedBatches: c.claimedBatches.Load(),
		ClaimedTimers:  c.claimedTimers.Load(),
		Renewed:        c.renewed.Load(),
		RenewRejected:  c.renewRejected.Load(),
	}
}
