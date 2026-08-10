package dgraphengine

import (
	"context"
	"time"
)

// DG_NetworkRTT is the declared PRIOR for round-trip time to a Dgraph cluster
// in the same datacenter (gRPC + RAFT group leader). It seeds planning before
// the first live probe; ProbeEngine replaces it with a measured EWMA once fresh
// samples exist. Adjust for cross-region deployments via WithNetworkRTT.
const DG_NetworkRTT = 2 * time.Millisecond

// Probe measures the current round-trip to the Dgraph cluster by timing a
// trivial read-only query (the same one HealthCheck uses). It implements
// [metaengine.Prober] so ProbeEngine can build a live RTT tracker that feeds
// Profile().NetworkRTT. Read-only transactions bypass RAFT, so this measures
// client→Dgraph reachability latency, not consensus commit latency.
func (e *dgraphEngine) Probe(ctx context.Context) (time.Duration, error) {
	start := time.Now()
	if _, err := e.client.NewTxn().Query(ctx, `{ health(func: uid(0x1)) { uid } }`); err != nil {
		return 0, err
	}

	return time.Since(start), nil
}
