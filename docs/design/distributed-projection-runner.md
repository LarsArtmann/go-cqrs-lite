# Design Spike: Distributed Projection Runner

**Status:** Raw Idea — No Design Yet
**Module:** New `distributed/` or extend `watermill/`

## Problem

Currently, projections run in a single process. For high-throughput systems, multiple nodes should share projection work — each node processes a subset of events, coordinated via leader election or partition assignment.

## Approaches

### 1. Leader-Election-Based (Active/Passive)

One node is the leader and runs all projections. Other nodes stand by. On leader failure, a follower takes over.

- **Pro:** Simple, no event reordering, deterministic
- **Con:** No horizontal scaling of projection throughput
- **Already partially implemented:** `DistributedRunner` with `LeaderElection` exists in the codebase

### 2. Partition-Based (Active/Active)

Events are partitioned by aggregate ID (consistent hashing). Each node owns a subset of partitions and processes only its events.

- **Pro:** Horizontal scaling, higher throughput
- **Con:** Cross-aggregate projections (SubscribeAll) need special handling; rebalancing on node join/leave is complex
- **Implementation:** Requires a partition-aware subscriber (like Kafka consumer groups)

### 3. Work-Queue-Based

Events are pushed to a work queue (Redis, NATS, SQS). Each node pulls and processes events independently.

- **Pro:** Natural load balancing, fault tolerance
- **Con:** At-least-once semantics require idempotent handlers; ordering per aggregate must be preserved

## Key Considerations

- **Ordering**: Per-aggregate ordering is critical for correctness. Cross-aggregate ordering is not.
- **Checkpointing**: Each partition/node needs its own checkpoint. The checkpoint store must support per-key checkpoints.
- **Rebalancing**: When a node joins/leaves, partitions must be reassigned. In-flight events must complete or be reassigned.
- **Exactly-once vs at-least-once**: Projection handlers must be idempotent regardless. The runner should guarantee at-least-once.

## Recommendation

**Single-machine is sufficient for now.** The existing `DistributedRunner` with `LeaderElection` provides active/passive failover. Active/active partitioning is a significant complexity increase that should wait for a concrete consumer need with measured single-machine bottleneck.
