package pgengine

import (
	"context"
	"time"
)

// PG_NetworkRTT is the declared PRIOR for round-trip time to a Postgres server
// in the same datacenter (or over a Unix socket). It seeds planning before the
// first live probe; ProbeEngine replaces it with a measured EWMA once fresh
// samples exist. Adjust for cross-region deployments via WithNetworkRTT.
const PG_NetworkRTT = 1 * time.Millisecond

// Probe measures the current round-trip to the Postgres server by timing a
// SELECT 1. It implements [metaengine.Prober] so ProbeEngine can build a live
// RTT tracker that feeds Profile().NetworkRTT. A failed probe returns the error
// and is dropped by the probe loop (never recorded), so a stalled connection
// cannot poison the EWMA.
func (e *pgEngine) Probe(ctx context.Context) (time.Duration, error) {
	start := time.Now()
	// SELECT 1 is the cheapest server-round-trip: it exercises connection
	// acquire + network + parse + plan + execute without touching data.
	if err := e.db.PingContext(ctx); err != nil {
		return 0, err
	}

	return time.Since(start), nil
}

// MeasureTransact times a real point-lookup query against meta_map, exercising
// the full read path (parse + plan + B-tree index seek + JSONB decode). It
// implements [metaengine.TransactMeasurer] so ProbeEngine can build a live
// per-read latency tracker that feeds Profile().NsPerRead. The query targets a
// sentinel key that never exists, so it always returns zero rows — the timing
// captures index traversal cost without depending on user data.
func (e *pgEngine) MeasureTransact(ctx context.Context) (time.Duration, error) {
	start := time.Now()
	_, err := e.conn().ExecContext(ctx,
		`SELECT value FROM meta_map WHERE collection = $1 AND key = $2 LIMIT 1`,
		"__probe", "__probe",
	)
	if err != nil {
		return 0, err
	}

	return time.Since(start), nil
}
