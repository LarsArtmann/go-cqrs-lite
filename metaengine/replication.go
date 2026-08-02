package metaengine

import "time"

// Replication declares how an engine's data propagates across process boundaries.
// Modeled on DDIA Chapter 5 (Replication).
//
// All CQRS read models are eventually consistent — the projection host has
// measurable lag between an event being appended and the projection folding
// it. The replication dimension captures whether that eventual consistency is
// bounded to a single process or spans multiple processes/nodes.
//
// The only strongly-consistent operation in CQRS is the event store's
// optimistic-concurrency append. Everything downstream is a stale cache.
// The planner's job is to pick the cheapest cache with the right scope.
type Replication string

const (
	// ReplicationNone means the engine is single-node: data stays in this
	// process. All current engines (Memory, SQLite, Pebble, DuckDB, Postgres)
	// are ReplicationNone. The zero value of Replication is "", which equals
	// ReplicationNone — so every EngineProfile that doesn't set the field
	// defaults to no replication automatically.
	ReplicationNone Replication = ""

	// ReplicationSingleLeader means writes go to one leader and propagate to
	// followers asynchronously (e.g. Postgres streaming replication). Reads
	// from followers may be stale by the replication lag.
	ReplicationSingleLeader Replication = "single-leader"

	// ReplicationMultiLeader means multiple nodes accept writes and reconcile
	// via consensus (e.g. CockroachDB, Spanner). Reads are strongly consistent
	// within a region but may lag across regions.
	ReplicationMultiLeader Replication = "multi-leader"

	// ReplicationLeaderless means any node can accept writes and they converge
	// via CRDT merge or read-repair (e.g. Iroh iroh-docs, Dynamo, Cassandra).
	// No coordination needed; convergence is guaranteed for monotonic operations.
	ReplicationLeaderless Replication = "leaderless"
)

// IsReplicated returns true if the engine's data crosses process boundaries
// via any replication mode. Convenience for planner filtering and diagnostics.
func (p EngineProfile) IsReplicated() bool {
	return p.Replication != ReplicationNone
}

// EffectiveReplicationLag returns the expected replication lag, defaulting to
// zero (no lag) when unset. For replicated engines, this represents the
// typical delay between a write on one node and it being visible on another.
// Used for diagnostics, NOT for latency estimation — staleness is not latency.
func (p EngineProfile) EffectiveReplicationLag() time.Duration {
	return p.ReplicationLag
}

// EffectiveNetworkRTT returns the round-trip time to reach this engine's data,
// defaulting to zero (in-process) when unset. For remote engines, this is the
// fixed per-query network overhead. Used by the cost estimator as an additive
// latency component: total_latency = compute_cost + NetworkRTT.
func (p EngineProfile) EffectiveNetworkRTT() time.Duration {
	return p.NetworkRTT
}
