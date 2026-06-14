# ADR-0018: Distributed Checkpointing for Projections

| Field   | Value        |
| ------- | ------------ |
| Date    | 2026-06-14   |
| Status  | Proposed     |
| Decider | Lars Artmann |

## Context

The `projection/` module supports replay+live event processing with checkpoint
tracking. Currently, each projection instance has its own `CheckpointStore`
(in-memory or SQL). When multiple instances of the same projection run across
different processes:

1. **Duplicate processing** — all instances process the same events
2. **Checkpoint divergence** — instances disagree on position
3. **No coordination** — no mechanism to distribute work

This is fine for single-process deployments but blocks horizontal scaling.

## Decision

**Distributed checkpointing is a consumer concern.** The library provides
the primitives; consumers implement coordination.

### Current Architecture (Unchanged)

```
Event Store → Journal → Projection Runner → Handler
                                    ↓
                             CheckpointStore (per-instance)
```

### Consumer Pattern for Distributed Projection

```go
// Option 1: Leader election (recommended for most cases)
// Only the leader runs the projection; followers standby.
leader := consensus.NewLeaderElection(config)
if leader.IsLeader() {
    runner.Run(ctx, projection)  // only one instance processes
}

// Option 2: Shared checkpoint with locking (for partitioned projections)
// Use SELECT FOR UPDATE or advisory locks on the checkpoint row.
store := sql.NewSQLCheckpointStore(db)
// Consumer wraps with distributed lock before checkpoint operations
```

### Why Not a Library Module?

1. **Coordination requires infrastructure** — needs Redis, etcd, or a consensus
   algorithm (Raft). Adding any of these as a dependency violates the library's
   minimal-dependency principle.
2. **Deployment-specific** — Kubernetes, Nomad, and bare metal have different
   coordination primitives. A library-level solution would be wrong for most
   deployments.
3. **`CheckpointStore` is already an interface** — consumers can implement a
   distributed-aware checkpoint store with ~50 lines of code.

### What the Library Provides

- `event.SeekableJournal` — position-based reads (essential for distributed
  checkpointing)
- `snapshot.SnapshotStore` — projection state snapshots for fast failover
- `projection.Runner` — the core runner, agnostic to single vs multi-instance

## Consequences

- **+** Library stays dependency-free
- **+** Consumers choose their coordination strategy (Raft, Redis, k8s leader election)
- **+** Works with existing CheckpointStore interface — no breaking changes
- **-** Consumers must implement coordination themselves
- **-** No built-in partitioning (consumers must shard by aggregate ID)

## Future Extensions

- `projection.DistributedRunner` wrapper with pluggable `LeaderElection` interface
- Built-in Redis checkpoint store (behind build tag or separate module)
- Partition-aware journal reader (distribute by aggregate ID hash)

## References

- [Kubernetes Leader Election](https://kubernetes.io/docs/concepts/architecture/leases/)
- `projection/runner.go` — current Runner implementation
- `event/seekable_journal.go` — position-based reads
